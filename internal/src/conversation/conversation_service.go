package conversation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/ws"
)

type ConversationService interface {
	ListCursor(ctx context.Context, q ListConversationQuery, t *tenant.Tenant) ([]*Conversation, int, string, bool, error)
	GetByID(ctx context.Context, tenantID int, id int) (*Conversation, error)
	Create(ctx context.Context, req CreateConversationRequest, t *tenant.Tenant) (*Conversation, error)
	Assign(ctx context.Context, tenantID int, id int, agentID int) error
	Unassign(ctx context.Context, tenantID int, id int) error
	Resolve(ctx context.Context, tenantID int, id int) error
	Reopen(ctx context.Context, tenantID int, id int) error
	MarkRead(ctx context.Context, tenantID int, id int) error
	GetTotalUnread(ctx context.Context, tenantID int) (int, error)
}

type conversationService struct {
	repo   ConversationRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
	notifier ws.Notifier
}

func NewConversationService(repo ConversationRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, notifier ...ws.Notifier) ConversationService {
	svc := &conversationService{repo: repo, db: db, cfg: cfg, logger: logger}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

func (s *conversationService) ListCursor(ctx context.Context, q ListConversationQuery, t *tenant.Tenant) ([]*Conversation, int, string, bool, error) {
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	var lastActivityTime *time.Time
	if q.LastActivity != "" {
		parsed, err := time.Parse(time.RFC3339, q.LastActivity)
		if err != nil {
			return nil, 0, "", false, fmt.Errorf("invalid last_activity: %w", err)
		}
		lastActivityTime = &parsed
	}

	convs, err := s.repo.ListCursor(ctx, s.db, t.ID, q.Limit+1, q.Status, q.ChannelID, q.AgentID, q.Search, lastActivityTime, q.LastID)
	if err != nil {
		return nil, 0, "", false, err
	}

	hasMore := len(convs) > q.Limit
	if hasMore {
		convs = convs[:q.Limit]
	}

	var lastID int
	var lastActivityStr string
	if len(convs) > 0 {
		last := convs[len(convs)-1]
		lastID = last.ID
		if last.LastActivity != nil {
			lastActivityStr = last.LastActivity.Format(time.RFC3339)
		}
	}

	return convs, lastID, lastActivityStr, hasMore, nil
}

func (s *conversationService) GetByID(ctx context.Context, tenantID int, id int) (*Conversation, error) {
	return s.repo.GetByID(ctx, s.db, tenantID, id)
}

func (s *conversationService) Create(ctx context.Context, req CreateConversationRequest, t *tenant.Tenant) (*Conversation, error) {
	conv := &Conversation{
		TenantID:  t.ID,
		Status:    "unassigned",
		ProfileID: req.ProfileID,
		ChannelID: req.ChannelID,
	}
	if req.AgentID > 0 {
		conv.Status = "assigned"
		conv.AgentID = &req.AgentID
	}

	id, err := s.repo.Create(ctx, s.db, conv)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	conv.ID = id
	conv.CreatedAt = time.Now()
	conv.UpdatedAt = time.Now()
	return conv, nil
}

func (s *conversationService) Assign(ctx context.Context, tenantID int, id int, agentID int) error {
	if err := s.repo.Assign(ctx, s.db, tenantID, id, agentID); err != nil {
		return err
	}
	s.notifyConversationUpdated(tenantID, id)
	return nil
}

func (s *conversationService) Unassign(ctx context.Context, tenantID int, id int) error {
	if err := s.repo.Unassign(ctx, s.db, tenantID, id); err != nil {
		return err
	}
	s.notifyConversationUpdated(tenantID, id)
	return nil
}

func (s *conversationService) Resolve(ctx context.Context, tenantID int, id int) error {
	if err := s.repo.UpdateStatus(ctx, s.db, tenantID, id, "resolved"); err != nil {
		return err
	}
	s.notifyConversationUpdated(tenantID, id)
	return nil
}

func (s *conversationService) Reopen(ctx context.Context, tenantID int, id int) error {
	if err := s.repo.UpdateStatus(ctx, s.db, tenantID, id, "unassigned"); err != nil {
		return err
	}
	s.notifyConversationUpdated(tenantID, id)
	return nil
}

func (s *conversationService) notifyConversationUpdated(tenantID int, conversationID int) {
	if s.notifier == nil {
		return
	}
	conv, err := s.repo.GetByID(context.Background(), s.db, tenantID, conversationID)
	if err != nil {
		s.logger.Error("failed to get conversation for notification: %v", err)
		return
	}
	s.notifier.Notify(tenantID, "conversation-updated", conv)
}

func (s *conversationService) MarkRead(ctx context.Context, tenantID int, id int) error {
	return s.repo.MarkRead(ctx, s.db, tenantID, id)
}

func (s *conversationService) GetTotalUnread(ctx context.Context, tenantID int) (int, error) {
	return s.repo.GetTotalUnread(ctx, s.db, tenantID)
}
