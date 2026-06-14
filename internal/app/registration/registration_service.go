package registration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/hash"
	"centrachannel/internal/utils/logger"
)

type RegistrationService interface {
	Onboard(ctx context.Context, req OnboardRequest) (*OnboardResponse, error)
}

type registrationService struct {
	repo   tenant.TenantRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewRegistrationService(repo tenant.TenantRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) RegistrationService {
	return &registrationService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *registrationService) Onboard(ctx context.Context, req OnboardRequest) (*OnboardResponse, error) {
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
		Domain:    req.Domain,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tenantID, err := s.repo.Create(ctx, tx, t)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant: %w", err)
	}
	t.ID = tenantID

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

	err = s.repo.AttachRole(ctx, tx, tenantID, userID, roleIDs["super_admin"])
	if err != nil {
		return nil, fmt.Errorf("failed to attach super admin role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &OnboardResponse{
		Tenant: t,
		Admin: &AdminUser{
			ID:        userID,
			FirstName: firstName,
			Email:     req.AdminEmail,
			Username:  username,
		},
	}, nil
}
