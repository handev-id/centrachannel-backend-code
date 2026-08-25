package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

	if data.Key.FromMe {
		if s.rdb != nil {
			exists, _ := s.rdb.Exists(ctx, "webhook_dedup:"+data.Key.ID).Result()
			if exists > 0 {
				s.logger.Debug("skip FromMe message %s: already sent from system", data.Key.ID)
				return nil
			}
		}
		return s.handleOutgoingMessageSync(ctx, payload, &data)
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

	p, err := s.profileRepo.GetByExternalIDAndChannelID(ctx, s.db, phone, ch.ID)
	if err != nil {
		p = &profile.Profile{
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

	text, attachment := extractMessageContent(data.Message, data.MessageType)

	now := time.Now()
	lastMsg := map[string]interface{}{
		"text":        text,
		"sender_type": "contact",
	}
	lastMsgJSON, _ := json.Marshal(lastMsg)

	conv, err := s.findConversation(ctx, tenantID, p.ID, ch.ID)
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
	if err := s.convRepo.UpdateLastMessage(ctx, s.db, tenantID, conv.ID, lastMsgJSON, nil); err != nil {
		s.logger.Error("failed to update last_message: %v", err)
	}

	msg := &message.Message{
		TenantID:         tenantID,
		Text:             text,
		Attachment:       attachment,
		Status:           "unread",
		SenderID:         0,
		SenderType:       "contact",
		WebhookMessageID: &data.Key.ID,
		ConversationID:   conv.ID,
	}

	mid, err := s.msgRepo.Create(ctx, s.db, msg)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	msg.ID = mid
	msg.CreatedAt = now
	msg.UpdatedAt = now

	s.notifyNewMessage(tenantID, msg)

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

	msgStatus := "sent"
	switch data.Status {
	case "DELIVERED", "DELIVERY_ACK":
		msgStatus = "delivered"
	case "READ", "PLAYED":
		msgStatus = "read"
	case "SENT":
		msgStatus = "sent"
	case "FAILED":
		msgStatus = "failed"
	default:
		return nil
	}

	if err := s.msgRepo.UpdateStatusByWebhookID(ctx, s.db, data.KeyID, msgStatus); err != nil {
		s.logger.Debug("no message found for webhook id %s: %v", data.KeyID, err)
	}

	return nil
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

	if s.notifier != nil {
		s.notifier.Notify(device.TenantID, "device-updated", map[string]interface{}{
			"id":     device.ID,
			"status": newStatus,
		})
	}

	return nil
}

func (s *webhookService) handleOutgoingMessageSync(ctx context.Context, payload *EvolutionWebhookPayload, data *EvolutionMessageUpsert) error {
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

	p, err := s.profileRepo.GetByExternalIDAndChannelID(ctx, s.db, phone, ch.ID)
	if err != nil {
		p = &profile.Profile{
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

	text, attachment := extractMessageContent(data.Message, data.MessageType)

	now := time.Now()
	lastMsg := map[string]interface{}{
		"text":        text,
		"sender_type": "user",
	}
	lastMsgJSON, _ := json.Marshal(lastMsg)

	conv, err := s.findConversation(ctx, tenantID, p.ID, ch.ID)
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
	if err := s.convRepo.UpdateLastMessage(ctx, s.db, tenantID, conv.ID, lastMsgJSON, nil); err != nil {
		s.logger.Error("failed to update last_message on outgoing sync: %v", err)
	}

	msg := &message.Message{
		TenantID:         tenantID,
		Text:             text,
		Attachment:       attachment,
		Status:           "sent",
		SenderID:         0,
		SenderType:       "user",
		WebhookMessageID: &data.Key.ID,
		ConversationID:   conv.ID,
	}

	mid, err := s.msgRepo.Create(ctx, s.db, msg)
	if err != nil {
		return fmt.Errorf("failed to create message: %w", err)
	}
	msg.ID = mid
	msg.CreatedAt = now
	msg.UpdatedAt = now

	s.notifyNewMessage(tenantID, msg)

	s.logger.Info("synced outgoing message from phone: device=%s, contact=%s, msg_id=%s", payload.Instance, phone, data.Key.ID)
	return nil
}

// kept here because it depends on conversation repo
func (s *webhookService) findConversation(ctx context.Context, tenantID, profileID, channelID int) (*conversation.Conversation, error) {
	return s.convRepo.FindOpenByProfileAndChannel(ctx, s.db, tenantID, profileID, channelID)
}

func (s *webhookService) notifyNewMessage(tenantID int, msg *message.Message) {
	if s.notifier == nil {
		return
	}
	s.notifier.Notify(tenantID, "new-message", msg)
}
