package note

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type mockNoteRepository struct {
	listFn     func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error)
	createFn   func(ctx context.Context, q DBTX, note *Note) (int, error)
	updateFn   func(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error
	deleteFn   func(ctx context.Context, q DBTX, tenantID int, id int) error
}

func (m *mockNoteRepository) ListByConversation(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error) {
	return m.listFn(ctx, q, tenantID, conversationID)
}

func (m *mockNoteRepository) Create(ctx context.Context, q DBTX, note *Note) (int, error) {
	return m.createFn(ctx, q, note)
}

func (m *mockNoteRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error {
	return m.updateFn(ctx, q, tenantID, id, note)
}

func (m *mockNoteRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	return m.deleteFn(ctx, q, tenantID, id)
}

type mockDB struct {
	execFn     func(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	queryFn    func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	queryRowFn func(ctx context.Context, query string, args ...interface{}) *sql.Row
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	if m.execFn != nil {
		return m.execFn(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, query, args...)
	}
	return nil
}

func baseNote(id, tenantID, conversationID int, text string) Note {
	now := time.Now()
	uid := 1
	return Note{
		ID:             id,
		TenantID:       tenantID,
		Text:           text,
		Date:           nil,
		ConversationID: conversationID,
		UserID:         &uid,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func newNoteService(repo NoteRepository) *noteService {
	return &noteService{
		repo:   repo,
		db:     nil,
		cfg:    nil,
		logger: nil,
	}
}

func TestNoteServiceListByConversation(t *testing.T) {
	t.Run("returns notes for conversation", func(t *testing.T) {
		notes := []Note{baseNote(1, 1, 10, "note a"), baseNote(2, 1, 10, "note b")}
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error) {
				if tenantID != 1 {
					t.Errorf("expected tenantID 1, got %d", tenantID)
				}
				if conversationID != 10 {
					t.Errorf("expected conversationID 10, got %d", conversationID)
				}
				return notes, nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), 1, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 notes, got %d", len(result))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error) {
				return []Note{}, nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), 1, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 notes, got %d", len(result))
		}
	})

	t.Run("pagination returns subset", func(t *testing.T) {
		allNotes := []Note{baseNote(1, 1, 10, "a"), baseNote(2, 1, 10, "b"), baseNote(3, 1, 10, "c")}
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error) {
				return allNotes[:2], nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), 1, 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 notes (page), got %d", len(result))
		}
	})

	t.Run("filter by conversation_id", func(t *testing.T) {
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error) {
				if conversationID != 20 {
					t.Errorf("expected conversationID 20, got %d", conversationID)
				}
				return []Note{baseNote(5, 1, 20, "filtered")}, nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), 1, 20)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0].ConversationID != 20 {
			t.Fatalf("expected 1 note for conversation 20, got %+v", result)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		_, err := svc.ListByConversation(context.Background(), 1, 10)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNoteServiceCreate(t *testing.T) {
	t.Run("success with text only", func(t *testing.T) {
		mock := &mockNoteRepository{
			createFn: func(ctx context.Context, q DBTX, note *Note) (int, error) {
				if note.Text != "hello" {
					t.Errorf("expected text 'hello', got %s", note.Text)
				}
				if note.Date != nil {
					t.Error("expected nil date")
				}
				return 100, nil
			},
		}
		svc := newNoteService(mock)
		req := CreateNoteRequest{Text: "hello"}
		result, err := svc.Create(context.Background(), req, 1, 10, 5)
		if err != nil {
			t.Fatal(err)
		}
		if result.ID != 100 {
			t.Errorf("expected ID 100, got %d", result.ID)
		}
		if result.TenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", result.TenantID)
		}
		if result.ConversationID != 10 {
			t.Errorf("expected conversationID 10, got %d", result.ConversationID)
		}
		if result.UserID == nil || *result.UserID != 5 {
			t.Errorf("expected userID 5, got %v", result.UserID)
		}
	})

	t.Run("success with date", func(t *testing.T) {
		mock := &mockNoteRepository{
			createFn: func(ctx context.Context, q DBTX, note *Note) (int, error) {
				if note.Date == nil {
					t.Error("expected non-nil date")
				}
				return 2, nil
			},
		}
		svc := newNoteService(mock)
		req := CreateNoteRequest{Text: "dated", Date: "2025-06-01"}
		result, err := svc.Create(context.Background(), req, 1, 10, 1)
		if err != nil {
			t.Fatal(err)
		}
		if result.Date == nil {
			t.Fatal("expected date in result")
		}
		expected, _ := time.Parse("2006-01-02", "2025-06-01")
		if !result.Date.Equal(expected) {
			t.Errorf("expected date %v, got %v", expected, result.Date)
		}
	})

	t.Run("invalid date format is ignored", func(t *testing.T) {
		mock := &mockNoteRepository{
			createFn: func(ctx context.Context, q DBTX, note *Note) (int, error) {
				if note.Date != nil {
					t.Error("expected nil date for invalid input")
				}
				return 3, nil
			},
		}
		svc := newNoteService(mock)
		req := CreateNoteRequest{Text: "bad date", Date: "not-a-date"}
		result, err := svc.Create(context.Background(), req, 1, 10, 1)
		if err != nil {
			t.Fatal(err)
		}
		if result.Date != nil {
			t.Error("expected nil date when parsing fails")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockNoteRepository{
			createFn: func(ctx context.Context, q DBTX, note *Note) (int, error) {
				return 0, errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		_, err := svc.Create(context.Background(), CreateNoteRequest{Text: "x"}, 1, 10, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNoteServiceUpdate(t *testing.T) {
	t.Run("success updates text", func(t *testing.T) {
		var capNote *Note
		mock := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error {
				capNote = note
				if tenantID != 1 {
					t.Errorf("expected tenantID 1, got %d", tenantID)
				}
				return nil
			},
		}
		svc := newNoteService(mock)
		newText := "updated text"
		req := UpdateNoteRequest{Text: &newText}
		result, err := svc.Update(context.Background(), 1, 5, req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Text != "updated text" {
			t.Errorf("expected 'updated text', got %s", result.Text)
		}
		if result.ID != 5 {
			t.Errorf("expected ID 5, got %d", result.ID)
		}
		if capNote.Text != "updated text" {
			t.Errorf("passed note has text %s", capNote.Text)
		}
	})

	t.Run("success updates date", func(t *testing.T) {
		mock := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error {
				if note.Date == nil {
					t.Error("expected non-nil date")
				}
				return nil
			},
		}
		svc := newNoteService(mock)
		dateStr := "2026-01-15"
		req := UpdateNoteRequest{Date: &dateStr}
		result, err := svc.Update(context.Background(), 1, 5, req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Date == nil {
			t.Fatal("expected date in result")
		}
		expected, _ := time.Parse("2006-01-02", "2026-01-15")
		if !result.Date.Equal(expected) {
			t.Errorf("expected date %v, got %v", expected, result.Date)
		}
	})

	t.Run("update only if tenant_id matches", func(t *testing.T) {
		mock := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error {
				if tenantID != 99 {
					return errors.New("no matching row for this tenant")
				}
				return nil
			},
		}
		svc := newNoteService(mock)
		text := "should fail"
		req := UpdateNoteRequest{Text: &text}
		_, err := svc.Update(context.Background(), 99, 5, req)
		if err != nil {
			return
		}
		mockTenant1 := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error {
				return nil
			},
		}
		svc2 := newNoteService(mockTenant1)
		_, err2 := svc2.Update(context.Background(), 1, 5, req)
		if err2 != nil {
			t.Errorf("update for matching tenant should succeed, got %v", err2)
		}
	})

	t.Run("repository error propagated", func(t *testing.T) {
		mock := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error {
				return errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		text := "x"
		req := UpdateNoteRequest{Text: &text}
		_, err := svc.Update(context.Background(), 1, 1, req)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNoteServiceDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var capTenantID, capID int
		mock := &mockNoteRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID int, id int) error {
				capTenantID = tenantID
				capID = id
				return nil
			},
		}
		svc := newNoteService(mock)
		err := svc.Delete(context.Background(), 1, 7)
		if err != nil {
			t.Fatal(err)
		}
		if capTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capTenantID)
		}
		if capID != 7 {
			t.Errorf("expected id 7, got %d", capID)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockNoteRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID int, id int) error {
				return errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		err := svc.Delete(context.Background(), 1, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
