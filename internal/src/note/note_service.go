package note

import (
	"context"
	"database/sql"
	"time"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type NoteService interface {
	ListByConversation(ctx context.Context, tenantID int, conversationID int) ([]Note, error)
	Create(ctx context.Context, req CreateNoteRequest, tenantID int, conversationID int, userID int) (*Note, error)
	Update(ctx context.Context, tenantID int, id int, req UpdateNoteRequest) (*Note, error)
	Delete(ctx context.Context, tenantID int, id int) error
}

type noteService struct {
	repo   NoteRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewNoteService(repo NoteRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) NoteService {
	return &noteService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *noteService) ListByConversation(ctx context.Context, tenantID int, conversationID int) ([]Note, error) {
	return s.repo.ListByConversation(ctx, s.db, tenantID, conversationID)
}

func (s *noteService) Create(ctx context.Context, req CreateNoteRequest, tenantID int, conversationID int, userID int) (*Note, error) {
	var date *time.Time
	if req.Date != "" {
		t, err := time.Parse("2006-01-02", req.Date)
		if err == nil {
			date = &t
		}
	}

	uid := userID
	note := &Note{
		TenantID:       tenantID,
		Text:           req.Text,
		Date:           date,
		ConversationID: conversationID,
		UserID:         &uid,
	}

	id, err := s.repo.Create(ctx, s.db, note)
	if err != nil {
		return nil, err
	}

	note.ID = id
	note.CreatedAt = time.Now()
	note.UpdatedAt = time.Now()
	return note, nil
}

func (s *noteService) Update(ctx context.Context, tenantID int, id int, req UpdateNoteRequest) (*Note, error) {
	existing := &Note{
		Text: "",
		Date: nil,
	}

	if req.Text != nil { existing.Text = *req.Text }
	if req.Date != nil {
		t, err := time.Parse("2006-01-02", *req.Date)
		if err == nil { existing.Date = &t }
	}

	if err := s.repo.Update(ctx, s.db, tenantID, id, existing); err != nil {
		return nil, err
	}

	existing.ID = id
	existing.UpdatedAt = time.Now()
	return existing, nil
}

func (s *noteService) Delete(ctx context.Context, tenantID int, id int) error {
	return s.repo.Delete(ctx, s.db, tenantID, id)
}
