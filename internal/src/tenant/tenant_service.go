package tenant

import (
	"context"
	"database/sql"
	"time"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type TenantService interface {
	GetByID(ctx context.Context, id int) (*Tenant, error)
	Update(ctx context.Context, tenant *Tenant) error
}

type tenantService struct {
	repo   TenantRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewTenantService(repo TenantRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) TenantService {
	return &tenantService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *tenantService) GetByID(ctx context.Context, id int) (*Tenant, error) {
	return s.repo.GetByID(ctx, s.db, id)
}

func (s *tenantService) Update(ctx context.Context, tenant *Tenant) error {
	tenant.UpdatedAt = time.Now()
	return s.repo.Update(ctx, s.db, tenant)
}
