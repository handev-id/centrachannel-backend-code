package webhook

import (
	"context"
	"database/sql"

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

func (s *webhookService) ProcessMetaEvent(ctx context.Context, payload *MetaWebhookPayload) error {
	switch payload.Object {
	case "instagram":
		return s.handleInstagramEvent(ctx, payload)
	case "page", "facebook":
		return s.handleFacebookEvent(ctx, payload)
	default:
		s.logger.Debug("unhandled meta object type: %s", payload.Object)
		return nil
	}
}
