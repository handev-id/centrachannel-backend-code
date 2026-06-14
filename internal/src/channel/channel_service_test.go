package channel

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type mockDB struct {
	queryFunc func(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(ctx, query, args...)
	}
	return nil, nil
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return nil
}

type mockChannelRepository struct {
	listFunc func(ctx context.Context, q DBTX) ([]Channel, error)
}

func (m *mockChannelRepository) List(ctx context.Context, q DBTX) ([]Channel, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, q)
	}
	return nil, nil
}

func TestChannelService_List(t *testing.T) {
	now := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	logo := json.RawMessage(`{"url":"http://example.com/logo.png"}`)

	t.Run("returns all channels with correct fields", func(t *testing.T) {
		expected := []Channel{
			{ID: 1, Name: "News Channel", Type: "news", Logo: logo, CreatedAt: now, UpdatedAt: now},
			{ID: 2, Name: "Sports Channel", Type: "sports", CreatedAt: now, UpdatedAt: now},
		}

		repo := &mockChannelRepository{
			listFunc: func(ctx context.Context, q DBTX) ([]Channel, error) {
				return expected, nil
			},
		}
		svc := NewChannelService(repo, nil)

		got, err := svc.List(context.Background())
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}

		if len(got) != len(expected) {
			t.Fatalf("expected %d channels, got %d", len(expected), len(got))
		}

		for i := range expected {
			if got[i].ID != expected[i].ID {
				t.Errorf("channel[%d].ID = %d, want %d", i, got[i].ID, expected[i].ID)
			}
			if got[i].Name != expected[i].Name {
				t.Errorf("channel[%d].Name = %s, want %s", i, got[i].Name, expected[i].Name)
			}
			if got[i].Type != expected[i].Type {
				t.Errorf("channel[%d].Type = %s, want %s", i, got[i].Type, expected[i].Type)
			}
			if !got[i].CreatedAt.Equal(expected[i].CreatedAt) {
				t.Errorf("channel[%d].CreatedAt = %v, want %v", i, got[i].CreatedAt, expected[i].CreatedAt)
			}
			if !got[i].UpdatedAt.Equal(expected[i].UpdatedAt) {
				t.Errorf("channel[%d].UpdatedAt = %v, want %v", i, got[i].UpdatedAt, expected[i].UpdatedAt)
			}
			if len(expected[i].Logo) > 0 && string(got[i].Logo) != string(expected[i].Logo) {
				t.Errorf("channel[%d].Logo = %s, want %s", i, string(got[i].Logo), string(expected[i].Logo))
			}
		}
	})

	t.Run("returns empty slice when no channels", func(t *testing.T) {
		repo := &mockChannelRepository{
			listFunc: func(ctx context.Context, q DBTX) ([]Channel, error) {
				return []Channel{}, nil
			},
		}
		svc := NewChannelService(repo, nil)

		got, err := svc.List(context.Background())
		if err != nil {
			t.Fatalf("List() returned error: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("expected 0 channels, got %d", len(got))
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		expectedErr := errors.New("db error")
		repo := &mockChannelRepository{
			listFunc: func(ctx context.Context, q DBTX) ([]Channel, error) {
				return nil, expectedErr
			},
		}
		svc := NewChannelService(repo, nil)

		_, err := svc.List(context.Background())
		if err != expectedErr {
			t.Errorf("expected error %v, got %v", expectedErr, err)
		}
	})
}
