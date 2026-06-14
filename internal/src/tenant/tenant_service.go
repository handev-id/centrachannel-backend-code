package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"centrachannel/config"
	"centrachannel/internal/utils/hash"
	"centrachannel/internal/utils/logger"
)

type TenantService interface {
	Onboard(ctx context.Context, req OnboardRequest) (*OnboardResponse, error)
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

func (s *tenantService) Onboard(ctx context.Context, req OnboardRequest) (*OnboardResponse, error) {
	tenantName := req.Name
	if req.CompanyName != "" {
		tenantName = req.CompanyName
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	tenant := &Tenant{
		Name:      tenantName,
		Domain:    req.Domain,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tenantID, err := s.repo.Create(ctx, tx, tenant)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}
	tenant.ID = tenantID

	roleNames := []string{"super_admin", "admin", "agent"}
	roleIDs := make(map[string]int, len(roleNames))
	for _, name := range roleNames {
		roleID, err := s.repo.CreateRole(ctx, tx, tenantID, name)
		if err != nil {
			return nil, fmt.Errorf("failed to create role %s: %w", name, err)
		}
		roleIDs[name] = roleID
	}

	hashedPassword, err := hash.HashPassword(req.AdminPassword)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	username := strings.Split(req.AdminEmail, "@")[0]
	firstName := username

	user := &User{
		FirstName: firstName,
		Username:  username,
		Email:     req.AdminEmail,
		Password:  hashedPassword,
		TenantID:  tenantID,
	}

	userID, err := s.repo.CreateUser(ctx, tx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create admin user: %w", err)
	}

	err = s.repo.AttachRole(ctx, tx, tenantID, userID, roleIDs["super_admin"])
	if err != nil {
		return nil, fmt.Errorf("failed to attach super admin role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &OnboardResponse{
		Tenant: tenant,
		Admin: &AdminUser{
			ID:        userID,
			FirstName: firstName,
			Email:     req.AdminEmail,
			Username:  username,
		},
	}, nil
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
