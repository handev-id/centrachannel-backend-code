package webhook

import (
	"context"
	"encoding/json"
	"time"

	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
)

func (s *webhookService) handleFacebookEvent(ctx context.Context, payload *MetaWebhookPayload) error {
	for _, entry := range payload.Entry {
		pageID := entry.ID

		t, err := s.tenantRepo.GetByMetaPageID(ctx, s.db, pageID)
		if err != nil || t == nil {
			s.logger.Error("tenant not found for facebook page id %s: %v", pageID, err)
			continue
		}

		ch, err := s.channelRepo.GetByType(ctx, s.db, "facebook")
		if err != nil {
			s.logger.Error("channel not found for type facebook: %v", err)
			continue
		}

		if err := s.processMetaMessaging(ctx, t.ID, ch.ID, "facebook", entry.Messaging); err != nil {
			return err
		}
	}
	return nil
}

func (s *webhookService) handleInstagramEvent(ctx context.Context, payload *MetaWebhookPayload) error {
	for _, entry := range payload.Entry {
		pageID := entry.ID

		t, err := s.tenantRepo.GetByMetaInstagramBusinessID(ctx, s.db, pageID)
		if err != nil || t == nil {
			s.logger.Error("tenant not found for instagram business id %s: %v", pageID, err)
			continue
		}

		ch, err := s.channelRepo.GetByType(ctx, s.db, "instagram")
		if err != nil {
			s.logger.Error("channel not found for type instagram: %v", err)
			continue
		}

		if err := s.processMetaMessaging(ctx, t.ID, ch.ID, "instagram", entry.Messaging); err != nil {
			return err
		}
	}
	return nil
}

func (s *webhookService) processMetaMessaging(ctx context.Context, tenantID, channelID int, channelType string, messages []MetaWebhookMessage) error {
	for _, msg := range messages {
		if msg.Message == nil || msg.Sender == nil {
			continue
		}

		externalID := msg.Sender.ID
		text := msg.Message.Text

		c, err := s.contactRepo.GetByPhone(ctx, s.db, tenantID, externalID)
		if err != nil {
			displayName := externalID
			c = &contact.Contact{
				TenantID:  tenantID,
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

		p, err := s.profileRepo.GetByExternalIDAndChannelID(ctx, s.db, externalID, channelID)
		if err != nil {
			displayName := externalID
			p = &profile.Profile{
				ExternalID:  externalID,
				DisplayName: &displayName,
				IsMain:      true,
				ContactID:   c.ID,
				ChannelID:   channelID,
			}
			pid, err := s.profileRepo.Create(ctx, s.db, p)
			if err != nil {
				s.logger.Error("failed to create profile for meta sender %s: %v", externalID, err)
				continue
			}
			p.ID = pid
		}

		conv, err := s.findConversation(ctx, tenantID, p.ID, channelID)
		if err != nil {
			conv = &conversation.Conversation{
				TenantID:  tenantID,
				Status:    "unassigned",
				ProfileID: p.ID,
				ChannelID: channelID,
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
			TenantID:         tenantID,
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

		s.notifyNewMessage(tenantID, msgRecord)

		lastMsg := map[string]interface{}{
			"text":        msgText,
			"sender_type": "contact",
		}
		lastMsgJSON, _ := json.Marshal(lastMsg)
		_ = s.convRepo.UpdateLastMessage(ctx, s.db, tenantID, conv.ID, lastMsgJSON, nil)

		s.logger.Info("meta webhook: tenant=%d, channel=%s, sender=%s, text=%s, msg_id=%s",
			tenantID, channelType, externalID, text, msg.Message.MID)
	}
	return nil
}
