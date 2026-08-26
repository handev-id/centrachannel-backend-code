package action

import (
	"context"
	"database/sql"

	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/logger"
)

type ActionService struct {
	convRepo   conversation.ConversationRepository
	msgRepo    message.MessageRepository
	profRepo   profile.ProfileRepository
	deviceRepo whatsapp_device.WhatsAppDeviceRepository
	client     whatsapp_device.WhatsAppClient
	db         *sql.DB
	logger     *logger.Logger
}

func NewActionService(
	convRepo conversation.ConversationRepository,
	msgRepo message.MessageRepository,
	profRepo profile.ProfileRepository,
	deviceRepo whatsapp_device.WhatsAppDeviceRepository,
	client whatsapp_device.WhatsAppClient,
	db *sql.DB,
	logger *logger.Logger,
) *ActionService {
	return &ActionService{
		convRepo:   convRepo,
		msgRepo:    msgRepo,
		profRepo:   profRepo,
		deviceRepo: deviceRepo,
		client:     client,
		db:         db,
		logger:     logger,
	}
}

func (s *ActionService) MarkConversationRead(ctx context.Context, tenantID int, conversationID int) error {
	if err := s.convRepo.MarkRead(ctx, s.db, tenantID, conversationID); err != nil {
		return err
	}

	go s.sendWhatsAppReadReceipt(tenantID, conversationID)

	return nil
}

func (s *ActionService) sendWhatsAppReadReceipt(tenantID int, conversationID int) {
	ctx := context.Background()

	conv, err := s.convRepo.GetByID(ctx, s.db, tenantID, conversationID)
	if err != nil || conv == nil || conv.Contact == nil {
		return
	}

	profile, err := s.profRepo.GetByID(ctx, s.db, conv.ProfileID)
	if err != nil || profile == nil || profile.LinkedDeviceWhatsappID == nil {
		return
	}

	device, err := s.deviceRepo.GetByWhatsappID(ctx, s.db, *profile.LinkedDeviceWhatsappID)
	if err != nil || device == nil {
		return
	}

	msgs, err := s.msgRepo.GetUnreadByConversation(ctx, s.db, tenantID, conversationID)
	if err != nil || len(msgs) == 0 {
		return
	}

	keys := make([]whatsapp_device.ReadMessageKey, 0, len(msgs))
	for _, m := range msgs {
		if m.WebhookMessageID != nil {
			keys = append(keys, whatsapp_device.ReadMessageKey{
				ID:        *m.WebhookMessageID,
				FromMe:    false,
				RemoteJid: profile.ExternalID,
			})
		}
	}

	if len(keys) > 0 {
		if err := s.client.MarkMessagesAsRead(ctx, device, keys); err != nil {
			s.logger.Warn("failed to mark messages as read on whatsapp: %v", err)
		}
	}

	if err := s.msgRepo.MarkReadByConversation(ctx, s.db, tenantID, conversationID); err != nil {
		s.logger.Warn("failed to mark messages as read in db: %v", err)
	}
}
