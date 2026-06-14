package conversation_tag

import (
	"context"
	"database/sql"
)

type ConversationTagService interface {
	ListByConversation(ctx context.Context, tenantID int, conversationID int) ([]ConversationTag, error)
	Attach(ctx context.Context, tenantID int, conversationID int, tagID int) error
	Detach(ctx context.Context, tenantID int, conversationID int, tagID int) error
}

type conversationTagService struct {
	repo ConversationTagRepository
	db   *sql.DB
}

func NewConversationTagService(repo ConversationTagRepository, db *sql.DB) ConversationTagService {
	return &conversationTagService{repo: repo, db: db}
}

func (s *conversationTagService) ListByConversation(ctx context.Context, tenantID int, conversationID int) ([]ConversationTag, error) {
	return s.repo.ListByConversation(ctx, s.db, tenantID, conversationID)
}

func (s *conversationTagService) Attach(ctx context.Context, tenantID int, conversationID int, tagID int) error {
	return s.repo.Attach(ctx, s.db, tenantID, conversationID, tagID)
}

func (s *conversationTagService) Detach(ctx context.Context, tenantID int, conversationID int, tagID int) error {
	return s.repo.Detach(ctx, s.db, tenantID, conversationID, tagID)
}
