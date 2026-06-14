package conversation

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/ws"
)

type ConversationService interface {
	List(ctx context.Context, q ListConversationQuery, t *tenant.Tenant) (*PaginatedResponse, error)
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
	repo     ConversationRepository
	db       *sql.DB
	cfg      *config.Config
	logger   *logger.Logger
	notifier ws.Notifier
}

func NewConversationService(repo ConversationRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, notifier ...ws.Notifier) ConversationService {
	svc := &conversationService{repo: repo, db: db, cfg: cfg, logger: logger}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

func (s *conversationService) List(ctx context.Context, q ListConversationQuery, t *tenant.Tenant) (*PaginatedResponse, error) {
	if q.Page < 1 { q.Page = 1 }
	if q.Limit < 1 || q.Limit > 100 { q.Limit = 20 }
	offset := (q.Page - 1) * q.Limit

	convs, total, err := s.repo.List(ctx, s.db, t.ID, q.Limit, offset, q.Status, q.ChannelID, q.AgentID, q.Search)
	if err != nil {
		return nil, err
	}

	lastPage := int(math.Ceil(float64(total) / float64(q.Limit)))
	from := offset + 1
	to := offset + len(convs)
	if to > total { to = total }
	if total == 0 { from = 0; to = 0 }

	meta := PaginationMeta{Total: total, PerPage: q.Limit, CurrentPage: q.Page, LastPage: lastPage, From: from, To: to}
	return &PaginatedResponse{Meta: meta, Data: convs}, nil
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
	err := s.repo.Assign(ctx, s.db, tenantID, id, agentID)
	if err == nil && s.notifier != nil {
		s.notifier.Notify(tenantID, "conversation:updated", map[string]interface{}{
			"id":       id,
			"action":   "assign",
			"agent_id": agentID,
		})
	}
	return err
}

func (s *conversationService) Unassign(ctx context.Context, tenantID int, id int) error {
	err := s.repo.Unassign(ctx, s.db, tenantID, id)
	if err == nil && s.notifier != nil {
		s.notifier.Notify(tenantID, "conversation:updated", map[string]interface{}{
			"id":     id,
			"action": "unassign",
		})
	}
	return err
}

func (s *conversationService) Resolve(ctx context.Context, tenantID int, id int) error {
	err := s.repo.UpdateStatus(ctx, s.db, tenantID, id, "resolved")
	if err == nil && s.notifier != nil {
		s.notifier.Notify(tenantID, "conversation:updated", map[string]interface{}{
			"id":     id,
			"action": "resolve",
		})
	}
	return err
}

func (s *conversationService) Reopen(ctx context.Context, tenantID int, id int) error {
	err := s.repo.UpdateStatus(ctx, s.db, tenantID, id, "unassigned")
	if err == nil && s.notifier != nil {
		s.notifier.Notify(tenantID, "conversation:updated", map[string]interface{}{
			"id":     id,
			"action": "reopen",
		})
	}
	return err
}

func (s *conversationService) MarkRead(ctx context.Context, tenantID int, id int) error {
	return s.repo.MarkRead(ctx, s.db, tenantID, id)
}

func (s *conversationService) GetTotalUnread(ctx context.Context, tenantID int) (int, error) {
	return s.repo.GetTotalUnread(ctx, s.db, tenantID)
}
