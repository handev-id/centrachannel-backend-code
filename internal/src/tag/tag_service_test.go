package tag

import (
	"context"
	"errors"
	"testing"
	"time"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type mockTagRepository struct {
	listFn     func(ctx context.Context, q DBTX, tenantID int) ([]Tag, error)
	getByIDFn  func(ctx context.Context, q DBTX, tenantID int, id int) (*Tag, error)
	createFn   func(ctx context.Context, q DBTX, tag *Tag) (int, error)
	updateFn   func(ctx context.Context, q DBTX, tenantID int, id int, tag *Tag) error
	deleteFn   func(ctx context.Context, q DBTX, tenantID int, id int) error
}

func (m *mockTagRepository) List(ctx context.Context, q DBTX, tenantID int) ([]Tag, error) {
	return m.listFn(ctx, q, tenantID)
}

func (m *mockTagRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Tag, error) {
	return m.getByIDFn(ctx, q, tenantID, id)
}

func (m *mockTagRepository) Create(ctx context.Context, q DBTX, tag *Tag) (int, error) {
	return m.createFn(ctx, q, tag)
}

func (m *mockTagRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, tag *Tag) error {
	return m.updateFn(ctx, q, tenantID, id, tag)
}

func (m *mockTagRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	return m.deleteFn(ctx, q, tenantID, id)
}

func baseTag(id, tenantID int, name string) Tag {
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
		tags := []Tag{baseTag(1, 1, "support"), baseTag(2, 1, "billing")}
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int) ([]Tag, error) {
				if tenantID != 1 {
					t.Errorf("expected tenantID 1, got %d", tenantID)
				}
				return tags, nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(result))
		}
	})

	t.Run("empty list", func(t *testing.T) {
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int) ([]Tag, error) {
				return []Tag{}, nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 tags, got %d", len(result))
		}
	})

	t.Run("pagination returns subset", func(t *testing.T) {
		allTags := []Tag{baseTag(1, 1, "a"), baseTag(2, 1, "b"), baseTag(3, 1, "c")}
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int) ([]Tag, error) {
				return allTags[:2], nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 tags (page), got %d", len(result))
		}
	})

	t.Run("search filter returns matching", func(t *testing.T) {
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int) ([]Tag, error) {
				return []Tag{baseTag(2, 1, "urgent")}, nil
			},
		}
		svc := newTagService(mock)
		result, err := svc.List(context.Background(), 1)
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 1 || result[0].Name != "urgent" {
			t.Fatalf("expected 1 tag named 'urgent', got %+v", result)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockTagRepository{
			listFn: func(ctx context.Context, q DBTX, tenantID int) ([]Tag, error) {
				return nil, errors.New("db error")
			},
		}
		svc := newTagService(mock)
		_, err := svc.List(context.Background(), 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestTagServiceCreate(t *testing.T) {
	t.Run("success with color", func(t *testing.T) {
		var saved *Tag
		mock := &mockTagRepository{
			createFn: func(ctx context.Context, q DBTX, tag *Tag) (int, error) {
				saved = tag
				return 42, nil
			},
		}
		svc := newTagService(mock)
		color := "#00ff00"
		req := CreateTagRequest{Name: "new tag", Color: &color}
		result, err := svc.Create(context.Background(), req, 1)
		if err != nil {
			t.Fatal(err)
		}
		if result.ID != 42 {
			t.Errorf("expected ID 42, got %d", result.ID)
		}
		if result.Name != "new tag" {
			t.Errorf("expected name 'new tag', got %s", result.Name)
		}
		if result.TenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", result.TenantID)
		}
		if result.Color == nil || *result.Color != "#00ff00" {
			t.Errorf("expected color #00ff00, got %v", result.Color)
		}
		if saved.TenantID != 1 || saved.Name != "new tag" {
			t.Errorf("saved tag has wrong fields: %+v", saved)
		}
	})

	t.Run("success with nil color", func(t *testing.T) {
		mock := &mockTagRepository{
			createFn: func(ctx context.Context, q DBTX, tag *Tag) (int, error) {
				if tag.Color != nil {
					t.Error("expected nil color")
				}
				return 1, nil
			},
		}
		svc := newTagService(mock)
		req := CreateTagRequest{Name: "no color"}
		result, err := svc.Create(context.Background(), req, 1)
		if err != nil {
			t.Fatal(err)
		}
		if result.Color != nil {
			t.Error("expected nil color in result")
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockTagRepository{
			createFn: func(ctx context.Context, q DBTX, tag *Tag) (int, error) {
				return 0, errors.New("db error")
			},
		}
		svc := newTagService(mock)
		_, err := svc.Create(context.Background(), CreateTagRequest{Name: "x"}, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestTagServiceUpdate(t *testing.T) {
	t.Run("success updates name", func(t *testing.T) {
		existing := baseTag(1, 1, "old name")
		mock := &mockTagRepository{
			getByIDFn: func(ctx context.Context, q DBTX, tenantID int, id int) (*Tag, error) {
				return &existing, nil
			},
			updateFn: func(ctx context.Context, q DBTX, tenantID int, id int, tag *Tag) error {
				if tag.Name != "new name" {
					t.Errorf("expected name 'new name', got %s", tag.Name)
				}
				return nil
			},
		}
		svc := newTagService(mock)
		newName := "new name"
		req := UpdateTagRequest{Name: &newName}
		result, err := svc.Update(context.Background(), 1, 1, req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Name != "new name" {
			t.Errorf("expected 'new name', got %s", result.Name)
		}
	})

	t.Run("success updates color only", func(t *testing.T) {
		existing := baseTag(1, 1, "keep name")
		mock := &mockTagRepository{
			getByIDFn: func(ctx context.Context, q DBTX, tenantID int, id int) (*Tag, error) {
				return &existing, nil
			},
			updateFn: func(ctx context.Context, q DBTX, tenantID int, id int, tag *Tag) error {
				if tag.Name != "keep name" {
					t.Errorf("expected name preserved, got %s", tag.Name)
				}
				return nil
			},
		}
		svc := newTagService(mock)
		newColor := "#0000ff"
		req := UpdateTagRequest{Color: &newColor}
		result, err := svc.Update(context.Background(), 1, 1, req)
		if err != nil {
			t.Fatal(err)
		}
		if result.Name != "keep name" {
			t.Errorf("expected name 'keep name', got %s", result.Name)
		}
	})

	t.Run("not found for wrong tenant", func(t *testing.T) {
		mock := &mockTagRepository{
			getByIDFn: func(ctx context.Context, q DBTX, tenantID int, id int) (*Tag, error) {
				return nil, errors.New("not found")
			},
		}
		svc := newTagService(mock)
		_, err := svc.Update(context.Background(), 99, 1, UpdateTagRequest{})
		if err == nil {
			t.Fatal("expected error for mismatch tenant")
		}
	})
}

func TestTagServiceDelete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var capTenantID, capID int
		mock := &mockTagRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID int, id int) error {
				capTenantID = tenantID
				capID = id
				return nil
			},
		}
		svc := newTagService(mock)
		err := svc.Delete(context.Background(), 1, 5)
		if err != nil {
			t.Fatal(err)
		}
		if capTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capTenantID)
		}
		if capID != 5 {
			t.Errorf("expected id 5, got %d", capID)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		mock := &mockTagRepository{
			deleteFn: func(ctx context.Context, q DBTX, tenantID int, id int) error {
				return errors.New("db error")
			},
		}
		svc := newTagService(mock)
		err := svc.Delete(context.Background(), 1, 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
