package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"centrachannel/config"
	"centrachannel/internal/messenger"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/ws"
)

type MessageService interface {
	ResolveQuote(ctx context.Context, tenantID, profileID, channelID uuid.UUID, quote *Quote) error
	UpdateStatusByProviderID(ctx context.Context, tenantID uuid.UUID, providerID, status string) error
	Create(ctx context.Context, msg *Message, lastAgentID *uuid.UUID) (uuid.UUID, error)
	ApplyMutation(ctx context.Context, tenantID, profileID, channelID uuid.UUID, mutation Mutation) (*Message, error)
	ListCursor(ctx context.Context, tenantID uuid.UUID, conversationID uuid.UUID, q ListMessageQuery) ([]*Message, uuid.UUID, bool, error)
	Send(ctx context.Context, req SendMessageRequest, tenantID uuid.UUID, conversationID uuid.UUID) (*Message, error)
	UpdateStatus(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, status string) error
	Retry(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, senderID uuid.UUID) (*Message, error)
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
	notifier    ws.Notifier
}

func NewMessageService(repo MessageRepository, convRepo conversation.ConversationRepository, profileRepo profile.ProfileRepository, channelRepo channel.ChannelRepository, tenantRepo tenant.TenantRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, rdb *redis.Client, notifier ...ws.Notifier) MessageService {
	svc := &messageService{repo: repo, convRepo: convRepo, profileRepo: profileRepo, channelRepo: channelRepo, tenantRepo: tenantRepo, db: db, cfg: cfg, logger: logger, rdb: rdb}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

func (s *messageService) ListCursor(ctx context.Context, tenantID uuid.UUID, conversationID uuid.UUID, q ListMessageQuery) ([]*Message, uuid.UUID, bool, error) {
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 50
	}

	msgs, err := s.repo.ListCursor(ctx, s.db, tenantID, conversationID, q.Limit+1, q.LastID)
	if err != nil {
		return nil, uuid.Nil, false, err
	}

	hasMore := len(msgs) > q.Limit
	if hasMore {
		msgs = msgs[:q.Limit]
	}

	cursorID := uuid.Nil
	if len(msgs) > 0 {
		cursorID = msgs[len(msgs)-1].ID
	}

	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	return msgs, cursorID, hasMore, nil
}

func (s *messageService) Send(ctx context.Context, req SendMessageRequest, tenantID uuid.UUID, conversationID uuid.UUID) (*Message, error) {
	text, attachment, err := normalizeMessageContent(req)
	if err != nil {
		return nil, err
	}
	req.Text, req.Attachment = text, attachment
	msg := &Message{
		Reactions:      []Reaction{},
		TenantID:       tenantID,
		Text:           req.Text,
		Attachment:     req.Attachment,
		Status:         "pending",
		SenderID:       req.SenderID,
		SenderType:     req.SenderType,
		ConversationID: conversationID,
	}
	if req.ReplyToID != uuid.Nil {
		target, err := s.repo.GetByID(ctx, s.db, tenantID, conversationID, req.ReplyToID)
		if err != nil {
			return nil, fmt.Errorf("reply target: %w", err)
		}
		if target == nil || target.WebhookMessageID == nil || *target.WebhookMessageID == "" {
			return nil, fmt.Errorf("reply target has no provider message ID")
		}
		conv, err := s.convRepo.GetByID(ctx, s.db, tenantID, conversationID)
		if err != nil || conv == nil {
			return nil, fmt.Errorf("reply conversation not found")
		}
		ch, err := s.channelRepo.GetByID(ctx, s.db, conv.ChannelID)
		if err != nil || ch == nil || ch.Type != "whatsapp" {
			return nil, fmt.Errorf("reply is supported on WhatsApp devices only")
		}
		msg.Reply = &Quote{MessageID: &target.ID, ProviderMessageID: *target.WebhookMessageID, Text: target.Text, Attachment: target.Attachment, SenderType: target.SenderType}
		if err := validateReplyTarget(msg.Reply); err != nil {
			return nil, err
		}
		msg.WebhookMessageReplyID = &msg.Reply.ProviderMessageID
	}

	var lastAgentID *uuid.UUID
	if req.SenderType == "user" {
		lastAgentID = &req.SenderID
	}
	if _, err := s.Create(ctx, msg, lastAgentID); err != nil {
		return nil, err
	}

	// Send to external platform via messenger (non-blocking)
	go func() {
		if err := s.deliverToExternal(tenantID, conversationID, msg); err != nil {
			s.logger.Error("failed to deliver message %s: %v", msg.ID, err)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if statusErr := s.repo.UpdateStatus(ctx, s.db, tenantID, msg.ID, "failed"); statusErr != nil {
				s.logger.Error("failed to persist delivery failure: %v", statusErr)
			} else if s.notifier != nil {
				s.notifier.Notify(tenantID, "message-status-updated", nil)
			}
		}
	}()

	return msg, nil
}

func (s *messageService) deliverToExternal(tenantID uuid.UUID, conversationID uuid.UUID, msg *Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	conv, err := s.convRepo.GetByID(ctx, s.db, tenantID, conversationID)
	if err != nil || conv == nil {
		return fmt.Errorf("get conversation for delivery: %v", err)
	}

	prof, err := s.profileRepo.GetByID(ctx, s.db, tenantID, conv.ProfileID)
	if err != nil || prof == nil {
		return fmt.Errorf("get profile for delivery: %v", err)
	}

	ch, err := s.channelRepo.GetByID(ctx, s.db, conv.ChannelID)
	if err != nil || ch == nil {
		return fmt.Errorf("get channel for delivery: %v", err)
	}

	// Look up tenant channel configuration (per-tenant credentials)
	t, err := s.tenantRepo.GetByID(ctx, s.db, tenantID)
	if err != nil || t == nil {
		return fmt.Errorf("get tenant configuration for delivery: %v", err)
	}

	settings, err := tenant.ParseSettings(t.Settings)
	if err != nil {
		s.logger.Error("failed to parse tenant settings: %v", err)
		return err
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
			return fmt.Errorf("profile has no linked WhatsApp device")
		}
		evoCfg := messenger.EvolutionConfig{
			APIURL:   s.cfg.EvolutionAPIURL,
			APIKey:   s.cfg.EvolutionAPIKey,
			DeviceID: *prof.LinkedDeviceWhatsappID,
		}
		if evoCfg.APIURL == "" {
			return fmt.Errorf("evolution API URL is not configured")
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
	if msg.Reply != nil {
		extMsg.Quoted = &messenger.QuotedMessage{ProviderMessageID: msg.Reply.ProviderMessageID, FromMe: msg.Reply.SenderType != "contact", Text: msg.Reply.Text, Attachment: msg.Reply.Attachment}
	}

	extID, err := sender.Send(extMsg)
	if err != nil {
		return err
	}

	if extID != "sent" {
		if err := s.repo.UpdateWebhookID(ctx, s.db, tenantID, msg.ID, extID); err != nil {
			return fmt.Errorf("persist provider message ID: %w", err)
		}

		if s.rdb != nil {
			s.rdb.Set(ctx, "webhook_dedup:"+extID, 1, 60*time.Second)
		}
	}
	if err := s.repo.UpdateStatus(ctx, s.db, tenantID, msg.ID, "sent"); err != nil {
		return err
	}
	if s.notifier != nil {
		stored, err := s.repo.GetByID(ctx, s.db, tenantID, msg.ConversationID, msg.ID)
		if err != nil {
			s.logger.Error("read sent message for content notification: %v", err)
		}
		if stored != nil && (stored.IsEdited || len(stored.Reactions) > 0) {
			s.notifier.Notify(tenantID, "message-updated", stored)
			s.notifier.Notify(tenantID, "conversation-updated", map[string]interface{}{"id": stored.ConversationID})
		}
		s.notifier.Notify(tenantID, "message-status-updated", nil)
	}
	return nil
}

func (s *messageService) UpdateStatus(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, status string) error {
	if err := s.repo.UpdateStatus(ctx, s.db, tenantID, id, status); err != nil {
		return err
	}
	if s.notifier != nil {
		s.notifier.Notify(tenantID, "message-status-updated", nil)
	}
	return nil
}

func (s *messageService) Retry(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, senderID uuid.UUID) (*Message, error) {
	msg, err := s.repo.RetryFailed(ctx, s.db, tenantID, id, senderID)
	if err != nil {
		return nil, fmt.Errorf("message cannot be retried: %w", err)
	}
	if s.notifier != nil {
		s.notifier.Notify(tenantID, "message-status-updated", nil)
	}
	go func() {
		if err := s.deliverToExternal(tenantID, msg.ConversationID, msg); err != nil {
			s.logger.Error("failed to retry message %s: %v", msg.ID, err)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.repo.UpdateStatus(ctx, s.db, tenantID, msg.ID, "failed"); err != nil {
				s.logger.Error("failed to persist retry failure: %v", err)
			} else if s.notifier != nil {
				s.notifier.Notify(tenantID, "message-status-updated", nil)
			}
		}
	}()
	return msg, nil
}

func (s *messageService) Create(ctx context.Context, msg *Message, lastAgentID *uuid.UUID) (uuid.UUID, error) {
	lastMessage := map[string]interface{}{"id": msg.ID, "text": msg.Text, "sender_type": msg.SenderType}
	if len(msg.Attachment) > 0 {
		lastMessage["attachment"] = msg.Attachment
	}
	preview, err := json.Marshal(lastMessage)
	if err != nil {
		return uuid.Nil, err
	}
	id, err := s.repo.Create(ctx, s.db, msg, preview, lastAgentID, s.convRepo)
	if err != nil || id == uuid.Nil {
		return id, err
	}
	msg.ID = id
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	if msg.UpdatedAt.IsZero() {
		msg.UpdatedAt = msg.CreatedAt
	}
	if msg.Reactions == nil {
		msg.Reactions = []Reaction{}
	}
	if s.notifier != nil {
		s.notifier.Notify(msg.TenantID, "new-message", msg)
	}
	return id, nil
}

func (s *messageService) ApplyMutation(ctx context.Context, tenantID, profileID, channelID uuid.UUID, mutation Mutation) (*Message, error) {
	if (mutation.Kind != "edit" && mutation.Kind != "reaction") || mutation.TargetProviderID == "" || mutation.EventAt.IsZero() || mutation.ActorKey == "" {
		return nil, fmt.Errorf("invalid message mutation")
	}
	updated, err := s.repo.ApplyMutation(ctx, s.db, tenantID, profileID, channelID, mutation)
	if err != nil {
		return nil, err
	}

	if updated != nil && s.notifier != nil {
		s.notifier.Notify(tenantID, "message-updated", updated)
		if mutation.Kind == "edit" {
			s.notifier.Notify(tenantID, "conversation-updated", map[string]interface{}{"id": updated.ConversationID})
		}
	}
	return updated, nil
}

func (s *messageService) ResolveQuote(ctx context.Context, tenantID, profileID, channelID uuid.UUID, quote *Quote) error {
	target, err := s.repo.GetByProviderID(ctx, s.db, tenantID, profileID, channelID, quote.ProviderMessageID)
	if errors.Is(err, ErrMessageNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("resolve quote: %w", err)
	}
	if target == nil {
		return nil
	}
	quote.MessageID = &target.ID
	quote.SenderType = target.SenderType
	var stored, snapshot Attachment
	if json.Unmarshal(target.Attachment, &stored) == nil && json.Unmarshal(quote.Attachment, &snapshot) == nil && stored.Type == snapshot.Type {
		snapshot.URL = stored.URL
		snapshot.Status = stored.Status
		snapshot.Size = stored.Size
		quote.Attachment, err = json.Marshal(snapshot)
	}
	return err
}

func (s *messageService) UpdateStatusByProviderID(ctx context.Context, tenantID uuid.UUID, providerID, status string) error {
	if err := s.repo.UpdateStatusByWebhookID(ctx, s.db, tenantID, providerID, status); err != nil {
		return err
	}
	if s.notifier != nil {
		s.notifier.Notify(tenantID, "message-status-updated", nil)
	}
	return nil
}
