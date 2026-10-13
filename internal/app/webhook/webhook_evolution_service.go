package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/utils/whatsapp"
)

func (s *webhookService) handleMessageUpsert(ctx context.Context, payload *EvolutionWebhookPayload) error {
	if payload.Data == nil || payload.Instance == "" {
		return nil
	}

	var data EvolutionMessageUpsert
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return fmt.Errorf("failed to parse messages.upsert data: %w", err)
	}
	// Preserve mediaKey/directPath and other provider fields needed for decryption.
	var original struct {
		Message json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(payload.Data, &original); err != nil {
		return err
	}
	data.RawMessage = original.Message
	if isEvolutionGroup(data.Key.RemoteJid) {
		return nil
	}

	if data.Key.FromMe {
		if s.rdb != nil {
			exists, _ := s.rdb.Exists(ctx, "webhook_dedup:"+data.Key.ID).Result()
			if exists > 0 {
				s.logger.Debug("skip FromMe message %s: already sent from system", data.Key.ID)
				return nil
			}
		}
	}

	device, err := s.deviceRepo.GetByWhatsappID(ctx, s.db, payload.Instance)
	if err != nil {
		return fmt.Errorf("device not found for instance %s: %w", payload.Instance, err)
	}

	tenantID := device.TenantID

	phone := whatsapp.ExtractPhoneFromJID(data.Key.RemoteJid)
	if phone == "" {
		return fmt.Errorf("invalid remote jid: %s", data.Key.RemoteJid)
	}

	ch, err := s.channelRepo.GetByType(ctx, s.db, "whatsapp")
	if err != nil {
		return fmt.Errorf("whatsapp channel not found: %w", err)
	}

	displayName := data.PushName
	if displayName == "" {
		displayName = phone
	}
	atPhone := "@" + phone

	parts := strings.SplitN(displayName, " ", 2)
	firstName := parts[0]
	var lastName *string
	if len(parts) > 1 {
		lastName = &parts[1]
	}

	c, err := s.contactRepo.GetByPhone(ctx, s.db, tenantID, phone)
	if err != nil {
		c = &contact.Contact{
			TenantID:  tenantID,
			FirstName: firstName,
			LastName:  lastName,
			Username:  &atPhone,
			Phone:     &phone,
			Whatsapp:  &phone,
			Status:    "individual",
		}
		cid, err := s.contactRepo.Create(ctx, s.db, c)
		if err != nil {
			return fmt.Errorf("failed to create contact: %w", err)
		}
		c.ID = cid
	}

	p, err := s.profileRepo.GetByExternalIDAndChannelID(ctx, s.db, tenantID, phone, ch.ID)
	if err != nil {
		p = &profile.Profile{
			TenantID:               tenantID,
			ExternalID:             phone,
			Username:               &phone,
			DisplayName:            &displayName,
			IsMain:                 true,
			LinkedDeviceWhatsappID: &payload.Instance,
			ContactID:              c.ID,
			ChannelID:              ch.ID,
		}
		pid, err := s.profileRepo.Create(ctx, s.db, p)
		if err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}
		p.ID = pid
	}

	if handled, err := s.applyEvolutionMutation(ctx, tenantID, p.ID, ch.ID, data.Key, data.Message, data.MessageTimestamp); handled {
		return err
	}
	text, attachment := extractMessageContent(data.Message, data.MessageType)
	if len(attachment) > 0 && s.media != nil {
		stored, err := s.media.Store(ctx, payload.Instance, &data, attachment)
		if err != nil {
			return fmt.Errorf("store incoming media: %w", err)
		}
		attachment = stored
	}

	conv, err := s.convRepo.FindOpenByProfileAndChannel(ctx, s.db, tenantID, p.ID, ch.ID)
	if err != nil {
		conv = &conversation.Conversation{
			TenantID:  tenantID,
			Status:    "unassigned",
			ProfileID: p.ID,
			ChannelID: ch.ID,
		}
		cid, err := s.convRepo.Create(ctx, s.db, conv)
		if err != nil {
			return fmt.Errorf("failed to create conversation: %w", err)
		}
		conv.ID = cid
	}

	senderType, status, senderID := "contact", "unread", c.ID
	if data.Key.FromMe {
		senderType, status, senderID = "user", "sent", uuid.Nil
	}
	msg := &message.Message{
		Reactions:        []message.Reaction{},
		TenantID:         tenantID,
		Text:             text,
		Attachment:       attachment,
		Status:           status,
		SenderID:         senderID,
		SenderType:       senderType,
		WebhookMessageID: &data.Key.ID,
		ConversationID:   conv.ID,
	}
	if err := s.populateEvolutionQuote(ctx, msg, data, p.ID, ch.ID, c); err != nil {
		return err
	}

	id, err := s.messages.Create(ctx, msg, nil)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	if id != uuid.Nil && data.Key.FromMe {
		s.logger.Info("synced outgoing message from phone: device=%s, contact=%s, msg_id=%s", payload.Instance, phone, data.Key.ID)
	}
	return nil
}

func (s *webhookService) handleMessageUpdate(ctx context.Context, payload *EvolutionWebhookPayload) error {
	if payload.Data == nil || payload.Instance == "" {
		return nil
	}

	var data EvolutionMessageUpdate
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return fmt.Errorf("failed to parse messages.update data: %w", err)
	}

	if data.Message != nil {
		content := unwrapEvolutionMessage(*data.Message)
		if content.EditedMessage != nil || content.ProtocolMessage != nil || content.SecretEncryptedMessage != nil {
			return s.handleEvolutionEditedUpdate(ctx, payload, data)
		}
	}
	msgStatus := "sent"
	switch data.Status {
	case "DELIVERED", "DELIVERY_ACK":
		msgStatus = "delivered"
	case "READ", "PLAYED":
		msgStatus = "read"
	case "SENT", "SERVER_ACK":
		msgStatus = "sent"
	case "FAILED":
		msgStatus = "failed"
	default:
		return nil
	}

	tenantID, err := s.resolveTenantFromInstance(ctx, payload.Instance)
	if err != nil {
		return err
	}
	if err := s.messages.UpdateStatusByProviderID(ctx, tenantID, data.KeyID, msgStatus); err != nil {
		s.logger.Debug("no message found for webhook id %s: %v", data.KeyID, err)
		return nil
	}

	return nil
}

func (s *webhookService) handleLegacyEditedMessage(ctx context.Context, payload *EvolutionWebhookPayload) error {
	var data EvolutionProtocolMessage
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return err
	}
	if data.EditedMessage == nil {
		return nil
	}
	return s.handleEvolutionEditedUpdate(ctx, payload, EvolutionMessageUpdate{
		KeyID: data.Key.ID, RemoteJid: data.Key.RemoteJid, FromMe: data.Key.FromMe, TimestampMs: data.TimestampMS,
		Message: &EvolutionMessage{EditedMessage: &EvolutionMessageContainer{Message: data.EditedMessage}},
	})
}

func (s *webhookService) handleConnectionUpdate(ctx context.Context, payload *EvolutionWebhookPayload) error {
	if payload.Data == nil || payload.Instance == "" {
		return nil
	}

	var data EvolutionConnectionUpdate
	if err := json.Unmarshal(payload.Data, &data); err != nil {
		return fmt.Errorf("failed to parse connection.update data: %w", err)
	}

	device, err := s.deviceRepo.GetByWhatsappID(ctx, s.db, payload.Instance)
	if err != nil {
		return fmt.Errorf("device not found for instance %s: %w", payload.Instance, err)
	}

	newStatus := "DISCONNECTED"
	switch data.State {
	case "open":
		newStatus = "CONNECTED"
	case "connecting":
		newStatus = "CONNECTING"
	}

	device.Status = newStatus
	if err := s.deviceRepo.Update(ctx, s.db, device.TenantID, device.ID, device); err != nil {
		return fmt.Errorf("failed to update device status: %w", err)
	}

	s.logger.Info("device %s status updated to %s via webhook", payload.Instance, newStatus)

	s.notifier.Notify(device.TenantID, "device-updated", map[string]interface{}{
		"id":     device.ID,
		"status": newStatus,
	})

	return nil
}

func (s *webhookService) resolveTenantFromInstance(ctx context.Context, instance string) (uuid.UUID, error) {
	device, err := s.deviceRepo.GetByWhatsappID(ctx, s.db, instance)
	if err != nil {
		return uuid.Nil, fmt.Errorf("device not found for instance %s: %w", instance, err)
	}
	if device == nil || device.TenantID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("device not found for instance %s", instance)
	}
	return device.TenantID, nil
}

func (s *webhookService) applyEvolutionMutation(ctx context.Context, tenantID, profileID, channelID uuid.UUID, key EvolutionMessageKey, content EvolutionMessage, timestamp EvolutionTimestamp) (bool, error) {
	event, handled, err := extractEvolutionMutation(key, content, timestamp)
	if err != nil || !handled {
		return handled, err
	}
	_, err = s.messages.ApplyMutation(ctx, tenantID, profileID, channelID, event)
	return true, err
}

func (s *webhookService) handleEvolutionEditedUpdate(ctx context.Context, payload *EvolutionWebhookPayload, data EvolutionMessageUpdate) error {
	if isEvolutionGroup(data.RemoteJid) {
		return nil
	}
	device, err := s.deviceRepo.GetByWhatsappID(ctx, s.db, payload.Instance)
	if err != nil {
		return err
	}
	if device == nil {
		return fmt.Errorf("device not found")
	}
	phone := whatsapp.ExtractPhoneFromJID(data.RemoteJid)
	if phone == "" {
		return fmt.Errorf("edited message has no peer")
	}
	channel, err := s.channelRepo.GetByType(ctx, s.db, "whatsapp")
	if err != nil {
		return err
	}
	profile, err := s.profileRepo.GetByExternalIDAndChannelID(ctx, s.db, device.TenantID, phone, channel.ID)
	if err != nil {
		return err
	}
	if profile == nil {
		return fmt.Errorf("edited message profile not found")
	}
	timestamp := data.MessageTimestamp
	if data.TimestampMs != 0 {
		timestamp = data.TimestampMs
	}
	_, err = s.applyEvolutionMutation(ctx, device.TenantID, profile.ID, channel.ID, EvolutionMessageKey{ID: data.KeyID, RemoteJid: data.RemoteJid, FromMe: data.FromMe}, *data.Message, timestamp)
	return err
}

func (s *webhookService) populateEvolutionQuote(ctx context.Context, msg *message.Message, data EvolutionMessageUpsert, profileID, channelID uuid.UUID, peer *contact.Contact) error {
	quote := extractEvolutionQuote(data.RawMessage, data.ContextInfo, data.Key)
	if quote == nil {
		return nil
	}
	if err := s.messages.ResolveQuote(ctx, msg.TenantID, profileID, channelID, quote); err != nil {
		return err
	}

	if quote.SenderType == "contact" {
		quote.SenderName = peer.FirstName
		if peer.LastName != nil {
			quote.SenderName += " " + *peer.LastName
		}
	}
	msg.Reply = quote
	msg.WebhookMessageReplyID = &quote.ProviderMessageID
	return nil
}
