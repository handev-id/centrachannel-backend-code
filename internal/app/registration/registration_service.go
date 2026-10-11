package registration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/src/domain"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/hash"
	"centrachannel/internal/utils/hostname"
	"centrachannel/internal/utils/logger"
	"centrachannel/internal/utils/tenantutils"
)

type RegistrationService interface {
	Onboard(ctx context.Context, req OnboardRequest) (*OnboardResponse, error)
}

type registrationService struct {
	repo    tenant.TenantRepository
	domains domain.DomainRepository
	db      *sql.DB
	cfg     *config.Config
	logger  *logger.Logger
}

func NewRegistrationService(repo tenant.TenantRepository, domains domain.DomainRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) RegistrationService {
	return &registrationService{repo: repo, domains: domains, db: db, cfg: cfg, logger: logger}
}

func (s *registrationService) Onboard(ctx context.Context, req OnboardRequest) (*OnboardResponse, error) {
	name, err := hostname.ValidateName(req.Domain, s.cfg.BaseDomain)
	if err != nil {
		return nil, err
	}
	tenantName := req.Name
	if req.CompanyName != "" {
		tenantName = req.CompanyName
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	t := &tenant.Tenant{
		Name:      tenantName,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tenantID, err := s.repo.Create(ctx, tx, t)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}
	t.ID = tenantID
	d := &domain.Domain{TenantID: tenantID, Name: name, IsDefault: true, IsActive: true}
	domainID, err := s.domains.Create(ctx, tx, d)
	if err != nil {
		return nil, fmt.Errorf("failed to create default domain: %w", err)
	}
	d.ID = domainID
	domainData, err := json.Marshal([]domain.Domain{*d})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize domain: %w", err)
	}

	roleNames := []string{"super-admin", "admin", "agent"}
	roleIDs := make(map[string]uuid.UUID, len(roleNames))
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

	user := &tenant.User{
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

	err = s.repo.AttachRole(ctx, tx, tenantID, userID, roleIDs["super-admin"])
	if err != nil {
		return nil, fmt.Errorf("failed to attach super admin role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	t.Settings = tenantutils.StripSettings(t.Settings)
	return &OnboardResponse{
		Tenant: &tenant.TenantResponse{Tenant: t, Domains: domainData},
		Admin: &AdminUser{
			ID:        userID,
			FirstName: firstName,
			Email:     req.AdminEmail,
			Username:  username,
		},
	}, nil
}
