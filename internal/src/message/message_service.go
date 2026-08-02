package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"centrachannel/config"
	"centrachannel/internal/messenger"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

type MessageService interface {
	ListCursor(ctx context.Context, conversationID int, q ListMessageQuery) ([]*Message, int, bool, error)
	Send(ctx context.Context, req SendMessageRequest, tenantID int, conversationID int) (*Message, error)
	UpdateStatus(ctx context.Context, id int, status string) error
}

type messageService struct {
	repo        MessageRepository
	convRepo    conversation.ConversationRepository
	profileRepo profile.ProfileRepository
	channelRepo channel.ChannelRepository
	tenantRepo  tenant.TenantRepository
	db          *sql.DB
	cfg         *config.Config
	logger      *logger.Logger
	rdb         *redis.Client
}

func NewMessageService(repo MessageRepository, convRepo conversation.ConversationRepository, profileRepo profile.ProfileRepository, channelRepo channel.ChannelRepository, tenantRepo tenant.TenantRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, rdb *redis.Client) MessageService {
	return &messageService{repo: repo, convRepo: convRepo, profileRepo: profileRepo, channelRepo: channelRepo, tenantRepo: tenantRepo, db: db, cfg: cfg, logger: logger, rdb: rdb}
}

func (s *messageService) ListCursor(ctx context.Context, conversationID int, q ListMessageQuery) ([]*Message, int, bool, error) {
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 50
	}

	msgs, err := s.repo.ListCursor(ctx, s.db, conversationID, q.Limit+1, q.LastID)
	if err != nil {
		return nil, 0, false, err
	}

	hasMore := len(msgs) > q.Limit
	if hasMore {
		msgs = msgs[:q.Limit]
	}

	cursorID := 0
	if len(msgs) > 0 {
		cursorID = msgs[len(msgs)-1].ID
	}

	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, cursorID, hasMore, nil
}

func (s *messageService) Send(ctx context.Context, req SendMessageRequest, tenantID int, conversationID int) (*Message, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	msg := &Message{
		TenantID:       tenantID,
		Text:           req.Text,
		Attachment:     req.Attachment,
		Status:         "sent",
		SenderID:       req.SenderID,
		SenderType:     req.SenderType,
		ConversationID: conversationID,
	}

	id, err := s.repo.Create(ctx, tx, msg)
	if err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	var lastMsgJSON []byte
	lastMsg := map[string]interface{}{
		"text":        req.Text,
		"sender_type": req.SenderType,
	}
	if len(req.Attachment) > 0 {
		var att map[string]interface{}
		if err := json.Unmarshal(req.Attachment, &att); err == nil {
			lastMsg["attachment"] = att
		}
	}
	lastMsgJSON, _ = json.Marshal(lastMsg)

	var lastAgentID *int
	if req.SenderType == "user" {
		lastAgentID = &req.SenderID
	}

	if err := s.convRepo.UpdateLastMessage(ctx, tx, tenantID, conversationID, lastMsgJSON, lastAgentID); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	msg.ID = id
	msg.CreatedAt = time.Now()
	msg.UpdatedAt = time.Now()

	// Send to external platform via messenger (non-blocking)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("panic in deliverToExternal: %v", r)
			}
		}()
		s.deliverToExternal(tenantID, conversationID, msg)
	}()

	return msg, nil
}

func (s *messageService) deliverToExternal(tenantID int, conversationID int, msg *Message) {
	ctx := context.Background()

	conv, err := s.convRepo.GetByID(ctx, s.db, tenantID, conversationID)
	if err != nil {
		s.logger.Error("failed to get conversation for external delivery: %v", err)
		return
	}

	prof, err := s.profileRepo.GetByID(ctx, s.db, conv.ProfileID)
	if err != nil {
		s.logger.Error("failed to get profile for external delivery: %v", err)
		return
	}

	ch, err := s.channelRepo.GetByID(ctx, s.db, conv.ChannelID)
	if err != nil {
		s.logger.Error("failed to get channel for external delivery: %v", err)
		return
	}

	// Look up tenant channel configuration (per-tenant credentials)
	t, err := s.tenantRepo.GetByID(ctx, s.db, tenantID)
	if err != nil {
		s.logger.Error("failed to get tenant config for external delivery: %v", err)
		return
	}

	settings, err := tenant.ParseSettings(t.Settings)
	if err != nil {
		s.logger.Error("failed to parse tenant settings: %v", err)
		return
	}

	var sender messenger.Messenger
	switch ch.Type {
	case "facebook", "instagram":
		cfg := messenger.MetaConfig{}
		if settings.ChannelConfiguration != nil {
			cfg = messenger.MetaConfig{
				AccessToken:     settings.ChannelConfiguration.MetaAccessToken,
				WhatsappPhoneID: settings.ChannelConfiguration.WhatsappPhoneID,
			}
		}
		sender = messenger.NewMetaSender(cfg, s.logger)
	case "whatsapp_business":
		instance := ""
		if settings.ChannelConfiguration != nil {
			instance = settings.ChannelConfiguration.EvolutionBusinessInstance
		}
		if instance != "" && s.cfg.EvolutionAPIURL != "" {
			evoCfg := messenger.EvolutionConfig{
				APIURL:   s.cfg.EvolutionAPIURL,
				APIKey:   s.cfg.EvolutionAPIKey,
				DeviceID: instance,
			}
			sender = messenger.NewEvolutionSender(evoCfg, s.logger)
		} else {
			cfg := messenger.MetaConfig{}
			if settings.ChannelConfiguration != nil {
				cfg = messenger.MetaConfig{
					AccessToken:     settings.ChannelConfiguration.MetaAccessToken,
					WhatsappPhoneID: settings.ChannelConfiguration.WhatsappPhoneID,
				}
			}
			sender = messenger.NewMetaSender(cfg, s.logger)
		}
	case "whatsapp":
		if prof.LinkedDeviceWhatsappID == nil || *prof.LinkedDeviceWhatsappID == "" {
			s.logger.Error("profile %d has no linked device for whatsapp delivery", prof.ID)
			return
		}
		evoCfg := messenger.EvolutionConfig{
			APIURL:   s.cfg.EvolutionAPIURL,
			APIKey:   s.cfg.EvolutionAPIKey,
			DeviceID: *prof.LinkedDeviceWhatsappID,
		}
		if evoCfg.APIURL == "" {
			s.logger.Warn("evolution api url not configured, falling back to mock sender")
			sender = messenger.NewMockSender()
		} else {
			sender = messenger.NewEvolutionSender(evoCfg, s.logger)
		}
	default:
		sender = messenger.NewMockSender()
	}

	extMsg := &messenger.OutgoingMessage{
		ChannelType: ch.Type,
		RecipientID: prof.ExternalID,
		Text:        msg.Text,
		Attachment:  msg.Attachment,
	}

	extID, err := sender.Send(extMsg)
	if err != nil {
		s.logger.Error("failed to deliver message to %s (conv=%d): %v", ch.Type, conversationID, err)
		_ = s.repo.UpdateStatus(ctx, s.db, msg.ID, "failed")
		return
	}

	if extID != "sent" {
		_ = s.repo.UpdateWebhookID(ctx, s.db, msg.ID, extID)

		if s.rdb != nil {
			s.rdb.Set(ctx, "webhook_dedup:"+extID, 1, 60*time.Second)
		}
	}
}

func (s *messageService) UpdateStatus(ctx context.Context, id int, status string) error {
	return s.repo.UpdateStatus(ctx, s.db, id, status)
}
