package note

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/ws"
)

type NoteService interface {
	ListByConversation(ctx context.Context, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error)
	Create(ctx context.Context, req CreateNoteRequest, tenantID uuid.UUID, conversationID uuid.UUID, userID uuid.UUID) (*Note, error)
	Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateNoteRequest) (*Note, error)
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type noteService struct {
	repo     NoteRepository
	db       *sql.DB
	cfg      *config.Config
	logger   *logger.Logger
	notifier ws.Notifier
}

func NewNoteService(repo NoteRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, notifier ...ws.Notifier) NoteService {
	svc := &noteService{repo: repo, db: db, cfg: cfg, logger: logger}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
}

func (s *noteService) ListByConversation(ctx context.Context, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
	return s.repo.ListByConversation(ctx, s.db, tenantID, conversationID)
}

func (s *noteService) Create(ctx context.Context, req CreateNoteRequest, tenantID uuid.UUID, conversationID uuid.UUID, userID uuid.UUID) (*Note, error) {
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

	s.notifier.Notify(tenantID, "note-created", note)

	return note, nil
}

func (s *noteService) Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateNoteRequest) (*Note, error) {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("note not found")
	}
	updated := *existing

	if req.Text != nil {
		updated.Text = *req.Text
	}
	if req.Date != nil {
		t, err := time.Parse("2006-01-02", *req.Date)
		if err != nil {
			return nil, fmt.Errorf("invalid date, use YYYY-MM-DD: %w", err)
		}
		updated.Date = &t
	}

	if err := s.repo.Update(ctx, s.db, tenantID, id, &updated); err != nil {
		return nil, err
	}

	updated.UpdatedAt = time.Now()

	s.notifier.Notify(tenantID, "note-updated", &updated)

	return &updated, nil
}

func (s *noteService) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	note := &Note{ID: id, TenantID: tenantID}

	if err := s.repo.Delete(ctx, s.db, tenantID, id); err != nil {
		return err
	}

	s.notifier.Notify(tenantID, "note-deleted", note)

	return nil
}
