package note

import (
	"context"
	"database/sql"
	"time"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/ws"
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
	notifier ws.Notifier
}

func NewNoteService(repo NoteRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, notifier ...ws.Notifier) NoteService {
	svc := &noteService{repo: repo, db: db, cfg: cfg, logger: logger}
	if len(notifier) > 0 {
		svc.notifier = notifier[0]
	}
	return svc
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

	s.notifyNoteEvent(tenantID, "note-created", note)

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

	s.notifyNoteEvent(tenantID, "note-updated", existing)

	return existing, nil
}

func (s *noteService) Delete(ctx context.Context, tenantID int, id int) error {
	note := &Note{ID: id, TenantID: tenantID}

	if err := s.repo.Delete(ctx, s.db, tenantID, id); err != nil {
		return err
	}

	s.notifyNoteEvent(tenantID, "note-deleted", note)

	return nil
}

func (s *noteService) notifyNoteEvent(tenantID int, event string, note *Note) {
	if s.notifier == nil {
		return
	}
	s.notifier.Notify(tenantID, event, note)
}
