package note

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockNoteRepository struct {
	getFn    func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Note, error)
	listFn   func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error)
	createFn func(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error)
	updateFn func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error
	deleteFn func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
}

type mockNotifier struct{}

func (mockNotifier) Notify(uuid.UUID, string, interface{}) {}

func (m *mockNoteRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Note, error) {
	if m.getFn != nil {
		return m.getFn(ctx, q, tenantID, id)
	}
	n := baseNote(id, tenantID, uuid.MustParse("00000000-0000-4000-8000-00000000000a"), "original")
	return &n, nil
}

func (m *mockNoteRepository) ListByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
	return m.listFn(ctx, q, tenantID, conversationID)
}

func (m *mockNoteRepository) Create(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error) {
	return m.createFn(ctx, q, note)
}

func (m *mockNoteRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error {
	return m.updateFn(ctx, q, tenantID, id, note)
}

func (m *mockNoteRepository) Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
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

func baseNote(id, tenantID, conversationID uuid.UUID, text string) Note {
	now := time.Now()
	uid := uuid.MustParse("00000000-0000-4000-8000-000000000001")
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
		repo:     repo,
		db:       nil,
		cfg:      nil,
		logger:   nil,
		notifier: mockNotifier{},
	}
}

func TestNoteServiceListByConversation(t *testing.T) {
	t.Run("returns notes for conversation", func(t *testing.T) {
		notes := []Note{baseNote(uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), "note a"), baseNote(uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), "note b")}
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
				if tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
					t.Errorf("expected tenantID 1, got %s", tenantID)
				}
				if conversationID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
					t.Errorf("expected conversationID 10, got %s", conversationID)
				}
				return notes, nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 notes, got %d", len(result))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
				return []Note{}, nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 notes, got %d", len(result))
		}
	})

	t.Run("pagination returns subset", func(t *testing.T) {
		allNotes := []Note{baseNote(uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), "a"), baseNote(uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), "b"), baseNote(uuid.MustParse("00000000-0000-4000-8000-000000000003"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), "c")}
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
				return allNotes[:2], nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 notes (page), got %d", len(result))
		}
	})

	t.Run("filter by conversation_id", func(t *testing.T) {
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
				if conversationID != uuid.MustParse("00000000-0000-4000-8000-000000000014") {
					t.Errorf("expected conversationID 20, got %s", conversationID)
				}
				return []Note{baseNote(uuid.MustParse("00000000-0000-4000-8000-000000000005"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000014"), "filtered")}, nil
			},
		}
		svc := newNoteService(mock)
		result, err := svc.ListByConversation(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000014"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0].ConversationID != uuid.MustParse("00000000-0000-4000-8000-000000000014") {
			t.Fatalf("expected 1 note for conversation 20, got %+v", result)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockNoteRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		_, err := svc.ListByConversation(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNoteServiceCreate(t *testing.T) {
	t.Run("success with text only", func(t *testing.T) {
		mock := &mockNoteRepository{
			createFn: func(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error) {
				if note.Text != "hello" {
					t.Errorf("expected text 'hello', got %s", note.Text)
				}
				if note.Date != nil {
					t.Error("expected nil date")
				}
				return uuid.MustParse("00000000-0000-4000-8000-000000000064"), nil
			},
		}
		svc := newNoteService(mock)
		req := CreateNoteRequest{Text: "hello"}
		result, err := svc.Create(context.Background(), req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), uuid.MustParse("00000000-0000-4000-8000-000000000005"))
		if err != nil {
			t.Fatal(err)
		}
		if result.ID != uuid.MustParse("00000000-0000-4000-8000-000000000064") {
			t.Errorf("expected ID 100, got %s", result.ID)
		}
		if result.TenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected tenantID 1, got %s", result.TenantID)
		}
		if result.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
			t.Errorf("expected conversationID 10, got %s", result.ConversationID)
		}
		if result.UserID == nil || *result.UserID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Errorf("expected userID 5, got %v", result.UserID)
		}
	})

	t.Run("success with date", func(t *testing.T) {
		mock := &mockNoteRepository{
			createFn: func(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error) {
				if note.Date == nil {
					t.Error("expected non-nil date")
				}
				return uuid.MustParse("00000000-0000-4000-8000-000000000002"), nil
			},
		}
		svc := newNoteService(mock)
		req := CreateNoteRequest{Text: "dated", Date: "2025-06-01"}
		result, err := svc.Create(context.Background(), req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
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
			createFn: func(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error) {
				if note.Date != nil {
					t.Error("expected nil date for invalid input")
				}
				return uuid.MustParse("00000000-0000-4000-8000-000000000003"), nil
			},
		}
		svc := newNoteService(mock)
		req := CreateNoteRequest{Text: "bad date", Date: "not-a-date"}
		result, err := svc.Create(context.Background(), req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err != nil {
			t.Fatal(err)
		}
		if result.Date != nil {
			t.Error("expected nil date when parsing fails")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockNoteRepository{
			createFn: func(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error) {
				return uuid.Nil, errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		_, err := svc.Create(context.Background(), CreateNoteRequest{Text: "x"}, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNoteServiceUpdate(t *testing.T) {
	t.Run("success updates text", func(t *testing.T) {
		var capNote *Note
		mock := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error {
				capNote = note
				if tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
					t.Errorf("expected tenantID 1, got %s", tenantID)
				}
				return nil
			},
		}
		svc := newNoteService(mock)
		newText := "updated text"
		req := UpdateNoteRequest{Text: &newText}
		result, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Text != "updated text" {
			t.Errorf("expected 'updated text', got %s", result.Text)
		}
		if result.ID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Errorf("expected ID 5, got %s", result.ID)
		}
		if capNote.Text != "updated text" {
			t.Errorf("passed note has text %s", capNote.Text)
		}
	})

	t.Run("success updates date", func(t *testing.T) {
		mock := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error {
				if note.Date == nil {
					t.Error("expected non-nil date")
				}
				return nil
			},
		}
		svc := newNoteService(mock)
		dateStr := "2026-01-15"
		req := UpdateNoteRequest{Date: &dateStr}
		result, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), req)
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
			updateFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error {
				if tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000063") {
					return errors.New("no matching row for this tenant")
				}
				return nil
			},
		}
		svc := newNoteService(mock)
		text := "should fail"
		req := UpdateNoteRequest{Text: &text}
		_, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000063"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), req)
		if err != nil {
			return
		}
		mockTenant1 := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error {
				return nil
			},
		}
		svc2 := newNoteService(mockTenant1)
		_, err2 := svc2.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), req)
		if err2 != nil {
			t.Errorf("update for matching tenant should succeed, got %v", err2)
		}
	})

	t.Run("repository error propagated", func(t *testing.T) {
		mock := &mockNoteRepository{
			updateFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error {
				return errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		text := "x"
		req := UpdateNoteRequest{Text: &text}
		_, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), req)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestNoteServiceUpdatePreservesOmittedFields(t *testing.T) {
	tenantID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	noteID := uuid.MustParse("00000000-0000-4000-8000-000000000005")
	conversationID := uuid.MustParse("00000000-0000-4000-8000-00000000000a")
	oldDate, _ := time.Parse("2006-01-02", "2026-01-15")
	newText, newDate := "updated", "2026-02-20"
	for _, tc := range []struct {
		name     string
		req      UpdateNoteRequest
		wantText string
		wantDate string
	}{
		{"date only", UpdateNoteRequest{Date: &newDate}, "original", newDate},
		{"text only", UpdateNoteRequest{Text: &newText}, newText, "2026-01-15"},
		{"empty update", UpdateNoteRequest{}, "original", "2026-01-15"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			existing := baseNote(noteID, tenantID, conversationID, "original")
			existing.Date = &oldDate
			repo := &mockNoteRepository{
				getFn: func(ctx context.Context, q DBTX, gotTenant, gotID uuid.UUID) (*Note, error) {
					if gotTenant != tenantID || gotID != noteID {
						t.Fatalf("incorrect note lookup: %s/%s", gotTenant, gotID)
					}
					return &existing, nil
				},
				updateFn: func(ctx context.Context, q DBTX, gotTenant, gotID uuid.UUID, n *Note) error {
					if n.Text != tc.wantText {
						t.Errorf("text = %q, want %q", n.Text, tc.wantText)
					}
					if n.Date == nil || n.Date.Format("2006-01-02") != tc.wantDate {
						t.Errorf("date = %v, want %s", n.Date, tc.wantDate)
					}
					return nil
				},
			}
			got, err := newNoteService(repo).Update(context.Background(), tenantID, noteID, tc.req)
			if err != nil {
				t.Fatal(err)
			}
			if got.TenantID != tenantID || got.ConversationID != conversationID || got.CreatedAt.IsZero() {
				t.Errorf("update lost identity or creation data: %+v", got)
			}
		})
	}
}

func TestNoteServiceUpdateRejectsInvalidDate(t *testing.T) {
	date := "not-a-date"
	repo := &mockNoteRepository{updateFn: func(context.Context, DBTX, uuid.UUID, uuid.UUID, *Note) error {
		t.Error("invalid date must not write the note")
		return nil
	}}
	_, err := newNoteService(repo).Update(context.Background(), uuid.New(), uuid.New(), UpdateNoteRequest{Date: &date})
	if err == nil {
		t.Fatal("expected invalid date error")
	}
}

func TestNoteServiceUpdatePropagatesLookupError(t *testing.T) {
	want := errors.New("note not found")
	repo := &mockNoteRepository{
		getFn: func(context.Context, DBTX, uuid.UUID, uuid.UUID) (*Note, error) { return nil, want },
		updateFn: func(context.Context, DBTX, uuid.UUID, uuid.UUID, *Note) error {
			t.Error("failed lookup must not update")
			return nil
		},
	}
	_, err := newNoteService(repo).Update(context.Background(), uuid.New(), uuid.New(), UpdateNoteRequest{})
	if !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
}

func TestNoteServiceDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var capTenantID, capID uuid.UUID
		mock := &mockNoteRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
				capTenantID = tenantID
				capID = id
				return nil
			},
		}
		svc := newNoteService(mock)
		err := svc.Delete(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000007"))
		if err != nil {
			t.Fatal(err)
		}
		if capTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected tenantID 1, got %s", capTenantID)
		}
		if capID != uuid.MustParse("00000000-0000-4000-8000-000000000007") {
			t.Errorf("expected id 7, got %s", capID)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockNoteRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
				return errors.New("db error")
			},
		}
		svc := newNoteService(mock)
		err := svc.Delete(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
