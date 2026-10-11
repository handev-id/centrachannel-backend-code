package tag

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type mockTagRepository struct {
	listFn    func(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error)
	getByIDFn func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Tag, error)
	createFn  func(ctx context.Context, q DBTX, tag *Tag) (uuid.UUID, error)
	updateFn  func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, tag *Tag) error
	deleteFn  func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
}

func (m *mockTagRepository) List(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error) {
	return m.listFn(ctx, q, tenantID)
}

func (m *mockTagRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Tag, error) {
	return m.getByIDFn(ctx, q, tenantID, id)
}

func (m *mockTagRepository) Create(ctx context.Context, q DBTX, tag *Tag) (uuid.UUID, error) {
	return m.createFn(ctx, q, tag)
}

func (m *mockTagRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, tag *Tag) error {
	return m.updateFn(ctx, q, tenantID, id, tag)
}

func (m *mockTagRepository) Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return m.deleteFn(ctx, q, tenantID, id)
}

func baseTag(id, tenantID uuid.UUID, name string) Tag {
	now := time.Now()
	color := "#ff0000"
	return Tag{
		ID:        id,
		TenantID:  tenantID,
		Name:      name,
		Color:     &color,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func newTagService(repo TagRepository) *tagService {
	return &tagService{
		repo:   repo,
		db:     nil,
		cfg:    &config.Config{},
		logger: logger.NewLogger("debug", "text"),
	}
}

func TestTagServiceList(t *testing.T) {
	t.Run("returns list of tags", func(t *testing.T) {
		tags := []Tag{baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "support"), baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "billing")}
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error) {
				if tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
					t.Errorf("expected tenantID 1, got %s", tenantID)
				}
				return tags, nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(result))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error) {
				return []Tag{}, nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 tags, got %d", len(result))
		}
	})

	t.Run("pagination returns subset", func(t *testing.T) {
		allTags := []Tag{baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "a"), baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "b"), baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000003"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "c")}
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error) {
				return allTags[:2], nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tags (page), got %d", len(result))
		}
	})

	t.Run("search filter returns matching", func(t *testing.T) {
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error) {
				return []Tag{baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000002"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "urgent")}, nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0].Name != "urgent" {
			t.Fatalf("expected 1 tag named 'urgent', got %+v", result)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newTagService(mock)
		_, err := svc.List(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestTagServiceCreate(t *testing.T) {
	t.Run("success with color", func(t *testing.T) {
		var saved *Tag
		mock := &mockTagRepository{
			createFn: func(ctx context.Context, q DBTX, tag *Tag) (uuid.UUID, error) {
				saved = tag
				return uuid.MustParse("00000000-0000-4000-8000-00000000002a"), nil
			},
		}
		svc := newTagService(mock)
		color := "#00ff00"
		req := CreateTagRequest{Name: "new tag", Color: &color}
		result, err := svc.Create(context.Background(), req, uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err != nil {
			t.Fatal(err)
		}
		if result.ID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
			t.Errorf("expected ID 42, got %s", result.ID)
		}
		if result.Name != "new tag" {
			t.Errorf("expected name 'new tag', got %s", result.Name)
		}
		if result.TenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected tenantID 1, got %s", result.TenantID)
		}
		if result.Color == nil || *result.Color != "#00ff00" {
			t.Errorf("expected color #00ff00, got %v", result.Color)
		}
		if saved.TenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") || saved.Name != "new tag" {
			t.Errorf("saved tag has wrong fields: %+v", saved)
		}
	})

	t.Run("success with nil color", func(t *testing.T) {
		mock := &mockTagRepository{
			createFn: func(ctx context.Context, q DBTX, tag *Tag) (uuid.UUID, error) {
				if tag.Color != nil {
					t.Error("expected nil color")
				}
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		svc := newTagService(mock)
		req := CreateTagRequest{Name: "no color"}
		result, err := svc.Create(context.Background(), req, uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err != nil {
			t.Fatal(err)
		}
		if result.Color != nil {
			t.Error("expected nil color in result")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockTagRepository{
			createFn: func(ctx context.Context, q DBTX, tag *Tag) (uuid.UUID, error) {
				return uuid.Nil, errors.New("db error")
			},
		}
		svc := newTagService(mock)
		_, err := svc.Create(context.Background(), CreateTagRequest{Name: "x"}, uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestTagServiceUpdate(t *testing.T) {
	t.Run("success updates name", func(t *testing.T) {
		existing := baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "old name")
		mock := &mockTagRepository{
			getByIDFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Tag, error) {
				return &existing, nil
			},
			updateFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, tag *Tag) error {
				if tag.Name != "new name" {
					t.Errorf("expected name 'new name', got %s", tag.Name)
				}
				return nil
			},
		}
		svc := newTagService(mock)
		newName := "new name"
		req := UpdateTagRequest{Name: &newName}
		result, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Name != "new name" {
			t.Errorf("expected 'new name', got %s", result.Name)
		}
	})

	t.Run("success updates color only", func(t *testing.T) {
		existing := baseTag(uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), "keep name")
		mock := &mockTagRepository{
			getByIDFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Tag, error) {
				return &existing, nil
			},
			updateFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, tag *Tag) error {
				if tag.Name != "keep name" {
					t.Errorf("expected name preserved, got %s", tag.Name)
				}
				return nil
			},
		}
		svc := newTagService(mock)
		newColor := "#0000ff"
		req := UpdateTagRequest{Color: &newColor}
		result, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Name != "keep name" {
			t.Errorf("expected name 'keep name', got %s", result.Name)
		}
	})

	t.Run("not found for wrong tenant", func(t *testing.T) {
		mock := &mockTagRepository{
			getByIDFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Tag, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTagService(mock)
		_, err := svc.Update(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000063"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), UpdateTagRequest{})
		if err == nil {
			t.Fatal("expected error for mismatch tenant")
		}
	})
}

func TestTagServiceDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var capTenantID, capID uuid.UUID
		mock := &mockTagRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
				capTenantID = tenantID
				capID = id
				return nil
			},
		}
		svc := newTagService(mock)
		err := svc.Delete(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"))
		if err != nil {
			t.Fatal(err)
		}
		if capTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected tenantID 1, got %s", capTenantID)
		}
		if capID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Errorf("expected id 5, got %s", capID)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockTagRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
				return errors.New("db error")
			},
		}
		svc := newTagService(mock)
		err := svc.Delete(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000001"))
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
