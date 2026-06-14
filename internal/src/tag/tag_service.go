package tag

import (
	"context"
	"database/sql"
	"time"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type TagService interface {
	List(ctx context.Context, tenantID int) ([]Tag, error)
	Create(ctx context.Context, req CreateTagRequest, tenantID int) (*Tag, error)
	Update(ctx context.Context, tenantID int, id int, req UpdateTagRequest) (*Tag, error)
	Delete(ctx context.Context, tenantID int, id int) error
}

type tagService struct {
	repo   TagRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewTagService(repo TagRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) TagService {
	return &tagService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *tagService) List(ctx context.Context, tenantID int) ([]Tag, error) {
	return s.repo.List(ctx, s.db, tenantID)
}

func (s *tagService) Create(ctx context.Context, req CreateTagRequest, tenantID int) (*Tag, error) {
	tag := &Tag{
		TenantID:  tenantID,
		Name:      req.Name,
		Color:     req.Color,
	}
	id, err := s.repo.Create(ctx, s.db, tag)
	if err != nil {
		return nil, err
	}
	tag.ID = id
	tag.CreatedAt = time.Now()
	tag.UpdatedAt = time.Now()
	return tag, nil
}

func (s *tagService) Update(ctx context.Context, tenantID int, id int, req UpdateTagRequest) (*Tag, error) {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil { existing.Name = *req.Name }
	if req.Color != nil { existing.Color = req.Color }

	if err := s.repo.Update(ctx, s.db, tenantID, id, existing); err != nil {
		return nil, err
	}

	existing.UpdatedAt = time.Now()
	return existing, nil
}

func (s *tagService) Delete(ctx context.Context, tenantID int, id int) error {
	return s.repo.Delete(ctx, s.db, tenantID, id)
}
