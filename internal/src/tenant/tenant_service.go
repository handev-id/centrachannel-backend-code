package tenant

import (
	"context"
	"database/sql"
	"encoding/json"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type TenantService interface {
	List(ctx context.Context) ([]Tenant, error)
	GetByID(ctx context.Context, id int) (*Tenant, error)
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

func (s *tenantService) List(ctx context.Context) ([]Tenant, error) {
	return s.repo.List(ctx, s.db)
}

func (s *tenantService) GetByID(ctx context.Context, id int) (*Tenant, error) {
	query := `SELECT id, name, domain, logo, address, phone, email, is_active, settings, created_at, updated_at FROM tenants WHERE id = $1`
	var t Tenant
	var logo, settings sql.NullString
	var address, phone, email sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID, &t.Name, &t.Domain, &logo, &address, &phone, &email, &t.IsActive, &settings, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if logo.Valid {
		t.Logo = json.RawMessage(logo.String)
	}
	if address.Valid {
		t.Address = &address.String
	}
	if phone.Valid {
		t.Phone = &phone.String
	}
	if email.Valid {
		t.Email = &email.String
	}
	if settings.Valid {
		t.Settings = json.RawMessage(settings.String)
	}

	return &t, nil
}
