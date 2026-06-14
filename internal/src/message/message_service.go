package message

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/ws"
)

type MessageService interface {
	List(ctx context.Context, conversationID int, q ListMessageQuery) (*PaginatedResponse, error)
	Send(ctx context.Context, req SendMessageRequest, tenantID int, conversationID int) (*Message, error)
	UpdateStatus(ctx context.Context, id int, status string) error
}

type messageService struct {
	repo         MessageRepository
	convRepo     conversation.ConversationRepository
	db           *sql.DB
	cfg          *config.Config
	logger       *logger.Logger
	notifier     ws.Notifier
}

func NewMessageService(repo MessageRepository, convRepo conversation.ConversationRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, notifier ...ws.Notifier) MessageService {
	svc := &messageService{repo: repo, convRepo: convRepo, db: db, cfg: cfg, logger: logger}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

func (s *messageService) List(ctx context.Context, conversationID int, q ListMessageQuery) (*PaginatedResponse, error) {
	if q.Page < 1 { q.Page = 1 }
	if q.Limit < 1 || q.Limit > 100 { q.Limit = 50 }
	offset := (q.Page - 1) * q.Limit

	msgs, total, err := s.repo.List(ctx, s.db, conversationID, q.Limit, offset)
	if err != nil {
		return nil, err
	}

	lastPage := int(math.Ceil(float64(total) / float64(q.Limit)))
	from := offset + 1
	to := offset + len(msgs)
	if to > total { to = total }
	if total == 0 { from = 0; to = 0 }

	meta := PaginationMeta{Total: total, PerPage: q.Limit, CurrentPage: q.Page, LastPage: lastPage, From: from, To: to}
	return &PaginatedResponse{Meta: meta, Data: msgs}, nil
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
		"sender_id":   req.SenderID,
		"created_at":  time.Now(),
	}
	if len(req.Attachment) > 0 {
		var att map[string]interface{}
		if err := json.Unmarshal(req.Attachment, &att); err == nil {
			lastMsg["attachment"] = att
		}
	}
	lastMsgJSON, _ = json.Marshal(lastMsg)

	lastAgentID := 0
	if req.SenderType == "user" {
		lastAgentID = req.SenderID
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

	if s.notifier != nil {
		s.notifier.Notify(tenantID, "message:new", msg)
	}

	return msg, nil
}

func (s *messageService) UpdateStatus(ctx context.Context, id int, status string) error {
	return s.repo.UpdateStatus(ctx, s.db, id, status)
}
