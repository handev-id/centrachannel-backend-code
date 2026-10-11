package registration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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
	repo   RegistrationRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewRegistrationService(repo RegistrationRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) RegistrationService {
	return &registrationService{repo: repo, db: db, cfg: cfg, logger: logger}
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

	t := &tenant.Tenant{
		Name:      tenantName,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	d := &domain.Domain{Name: name, IsDefault: true, IsActive: true}

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
	}

	userID, err := s.repo.Create(ctx, s.db, t, d, user, []string{"super-admin", "admin", "agent"}, "super-admin")
	if err != nil {
		return nil, err
	}
	domainData, err := json.Marshal([]domain.Domain{*d})
	if err != nil {
		return nil, fmt.Errorf("failed to serialize domain: %w", err)
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
