package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/ws"
)

type WebhookService interface {
	ProcessEvolutionEvent(ctx context.Context, payload *EvolutionWebhookPayload) error
	ProcessMetaEvent(ctx context.Context, payload *MetaWebhookPayload) error
}

type webhookService struct {
	deviceRepo  whatsapp_device.WhatsAppDeviceRepository
	contactRepo contact.ContactRepository
	profileRepo profile.ProfileRepository
	channelRepo channel.ChannelRepository
	convRepo    conversation.ConversationRepository
	msgRepo     message.MessageRepository
	tenantRepo  tenant.TenantRepository
	db          *sql.DB
	logger      *logger.Logger
	notifier    ws.Notifier
	rdb         *redis.Client
}

func NewWebhookService(deviceRepo whatsapp_device.WhatsAppDeviceRepository, contactRepo contact.ContactRepository, profileRepo profile.ProfileRepository, channelRepo channel.ChannelRepository, convRepo conversation.ConversationRepository, msgRepo message.MessageRepository, tenantRepo tenant.TenantRepository, db *sql.DB, logger *logger.Logger, rdb *redis.Client, notifier ...ws.Notifier) WebhookService {
	svc := &webhookService{
		deviceRepo:  deviceRepo,
		contactRepo: contactRepo,
		profileRepo: profileRepo,
		channelRepo: channelRepo,
		convRepo:    convRepo,
		msgRepo:     msgRepo,
		tenantRepo:  tenantRepo,
		db:          db,
		logger:      logger,
		rdb:         rdb,
	}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

func (s *webhookService) ProcessEvolutionEvent(ctx context.Context, payload *EvolutionWebhookPayload) error {
	switch payload.Event {
	case "messages.upsert":
		return s.handleMessageUpsert(ctx, payload)
	case "messages.update":
		return s.handleMessageUpdate(ctx, payload)
	case "connection.update":
		return s.handleConnectionUpdate(ctx, payload)
	default:
		s.logger.Debug("unhandled evolution event: %s", payload.Event)
		return nil
	}
}

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

	phone := extractPhoneFromJID(data.Key.RemoteJid)
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

	text, attachment := s.extractMessageContent(data.Message, data.MessageType)

	now := time.Now()
	lastMsg := map[string]interface{}{
		"text":        text,
		"sender_type": "contact",
		"created_at":  now,
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
	switch data.Update.Status {
	case "DELIVERED":
		msgStatus = "delivered"
	case "READ":
		msgStatus = "read"
	case "SENT":
		msgStatus = "sent"
	case "FAILED":
		msgStatus = "failed"
	default:
		return nil
	}

	if err := s.msgRepo.UpdateStatusByWebhookID(ctx, s.db, data.Key.ID, msgStatus); err != nil {
		s.logger.Debug("no message found for webhook id %s: %v", data.Key.ID, err)
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

	phone := extractPhoneFromJID(data.Key.RemoteJid)
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

	text, attachment := s.extractMessageContent(data.Message, data.MessageType)

	now := time.Now()
	lastMsg := map[string]interface{}{
		"text":        text,
		"sender_type": "user",
		"created_at":  now,
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

	s.logger.Info("synced outgoing message from phone: device=%s, contact=%s, msg_id=%s", payload.Instance, phone, data.Key.ID)
	return nil
}

func (s *webhookService) findConversation(ctx context.Context, tenantID, profileID, channelID int) (*conversation.Conversation, error) {
	return s.convRepo.FindOpenByProfileAndChannel(ctx, s.db, tenantID, profileID, channelID)
}

func (s *webhookService) extractMessageContent(evtMsg EvolutionMessage, msgType string) (*string, json.RawMessage) {
	if evtMsg.Conversation != nil && *evtMsg.Conversation != "" {
		return evtMsg.Conversation, nil
	}
	if evtMsg.ExtendedTextMessage != nil && evtMsg.ExtendedTextMessage.Text != "" {
		return &evtMsg.ExtendedTextMessage.Text, nil
	}

	if evtMsg.ImageMessage != nil && evtMsg.ImageMessage.URL != "" {
		att := map[string]interface{}{
			"url":      evtMsg.ImageMessage.URL,
			"type":     "image",
			"mimetype": evtMsg.ImageMessage.Mimetype,
		}
		if evtMsg.ImageMessage.Caption != "" {
			att["caption"] = evtMsg.ImageMessage.Caption
		}
		attJSON, _ := json.Marshal(att)
		caption := evtMsg.ImageMessage.Caption
		if caption == "" {
			return nil, attJSON
		}
		return &caption, attJSON
	}

	if evtMsg.VideoMessage != nil && evtMsg.VideoMessage.URL != "" {
		att := map[string]interface{}{
			"url":      evtMsg.VideoMessage.URL,
			"type":     "video",
			"mimetype": evtMsg.VideoMessage.Mimetype,
		}
		if evtMsg.VideoMessage.Caption != "" {
			att["caption"] = evtMsg.VideoMessage.Caption
		}
		attJSON, _ := json.Marshal(att)
		caption := evtMsg.VideoMessage.Caption
		if caption == "" {
			return nil, attJSON
		}
		return &caption, attJSON
	}

	if evtMsg.AudioMessage != nil && evtMsg.AudioMessage.URL != "" {
		att := map[string]interface{}{
			"url":      evtMsg.AudioMessage.URL,
			"type":     "audio",
			"mimetype": evtMsg.AudioMessage.Mimetype,
		}
		attJSON, _ := json.Marshal(att)
		return nil, attJSON
	}

	if evtMsg.DocumentMessage != nil && evtMsg.DocumentMessage.URL != "" {
		att := map[string]interface{}{
			"url":       evtMsg.DocumentMessage.URL,
			"type":      "document",
			"mimetype":  evtMsg.DocumentMessage.Mimetype,
			"file_name": evtMsg.DocumentMessage.FileName,
		}
		attJSON, _ := json.Marshal(att)
		return nil, attJSON
	}

	return nil, nil
}

func (s *webhookService) ProcessMetaEvent(ctx context.Context, payload *MetaWebhookPayload) error {
	channelType := "facebook"
	if payload.Object == "instagram" {
		channelType = "instagram"
	}

	for _, entry := range payload.Entry {
		pageID := entry.ID

		var t *tenant.Tenant
		var err error
		if channelType == "instagram" {
			t, err = s.tenantRepo.GetByMetaInstagramBusinessID(ctx, s.db, pageID)
		} else {
			t, err = s.tenantRepo.GetByMetaPageID(ctx, s.db, pageID)
		}
		if err != nil {
			s.logger.Error("tenant not found for %s page/id %s: %v", channelType, pageID, err)
			continue
		}

		ch, err := s.channelRepo.GetByType(ctx, s.db, channelType)
		if err != nil {
			s.logger.Error("channel not found for type %s: %v", channelType, err)
			continue
		}

		for _, msg := range entry.Messaging {
			if msg.Message == nil || msg.Sender == nil {
				continue
			}

			externalID := msg.Sender.ID
			text := msg.Message.Text

			c, err := s.contactRepo.GetByPhone(ctx, s.db, t.ID, externalID)
			if err != nil {
				displayName := externalID
				c = &contact.Contact{
					TenantID:  t.ID,
					FirstName: displayName,
					Status:    "individual",
				}
				if channelType == "facebook" {
					c.Facebook = &externalID
				} else {
					c.Instagram = &externalID
				}
				cid, err := s.contactRepo.Create(ctx, s.db, c)
				if err != nil {
					s.logger.Error("failed to create contact for meta sender %s: %v", externalID, err)
					continue
				}
				c.ID = cid
			}

			p, err := s.profileRepo.GetByExternalIDAndChannelID(ctx, s.db, externalID, ch.ID)
			if err != nil {
				displayName := externalID
				p = &profile.Profile{
					ExternalID:  externalID,
					DisplayName: &displayName,
					IsMain:      true,
					ContactID:   c.ID,
					ChannelID:   ch.ID,
				}
				pid, err := s.profileRepo.Create(ctx, s.db, p)
				if err != nil {
					s.logger.Error("failed to create profile for meta sender %s: %v", externalID, err)
					continue
				}
				p.ID = pid
			}

			conv, err := s.findConversation(ctx, t.ID, p.ID, ch.ID)
			if err != nil {
				conv = &conversation.Conversation{
					TenantID:  t.ID,
					Status:    "unassigned",
					ProfileID: p.ID,
					ChannelID: ch.ID,
				}
				cid, err := s.convRepo.Create(ctx, s.db, conv)
				if err != nil {
					s.logger.Error("failed to create conversation for meta sender %s: %v", externalID, err)
					continue
				}
				conv.ID = cid
			}

			msgText := &text
			if text == "" {
				msgText = nil
			}

			msgRecord := &message.Message{
				TenantID:         t.ID,
				Text:             msgText,
				Status:           "unread",
				SenderID:         0,
				SenderType:       "contact",
				WebhookMessageID: &msg.Message.MID,
				ConversationID:   conv.ID,
			}

			mid, err := s.msgRepo.Create(ctx, s.db, msgRecord)
			if err != nil {
				s.logger.Error("failed to create message for meta sender %s: %v", externalID, err)
				continue
			}
			msgRecord.ID = mid
			msgRecord.CreatedAt = time.Now()
			msgRecord.UpdatedAt = time.Now()

			lastMsg := map[string]interface{}{
				"text":        msgText,
				"sender_type": "contact",
				"created_at":  time.Now(),
			}
			lastMsgJSON, _ := json.Marshal(lastMsg)
			_ = s.convRepo.UpdateLastMessage(ctx, s.db, t.ID, conv.ID, lastMsgJSON, nil)

			s.logger.Info("meta webhook: tenant=%d, channel=%s, sender=%s, text=%s, msg_id=%s",
				t.ID, channelType, externalID, text, msg.Message.MID)
		}
	}
	return nil
}

func extractPhoneFromJID(jid string) string {
	if idx := strings.Index(jid, "@"); idx != -1 {
		return jid[:idx]
	}
	return jid
}
