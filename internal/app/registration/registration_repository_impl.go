package registration

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"centrachannel/internal/src/domain"
	"centrachannel/internal/src/tenant"
)

type registrationRepository struct {
	tenants tenant.TenantRepository
	domains domain.DomainRepository
}

func NewRegistrationRepository(tenants tenant.TenantRepository, domains domain.DomainRepository) RegistrationRepository {
	return &registrationRepository{tenants: tenants, domains: domains}
}

func (r *registrationRepository) Create(ctx context.Context, db *sql.DB, t *tenant.Tenant, d *domain.Domain, user *tenant.User, roleNames []string, adminRoleName string) (uuid.UUID, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()
	tenantID, err := r.tenants.Create(ctx, tx, t)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create tenant: %w", err)
	}
	t.ID = tenantID
	d.TenantID = tenantID
	domainID, err := r.domains.Create(ctx, tx, d)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create default domain: %w", err)
	}
	d.ID = domainID
	var adminRoleID uuid.UUID
	for _, name := range roleNames {
		roleID, err := r.tenants.CreateRole(ctx, tx, tenantID, name)
		if err != nil {
			return uuid.Nil, fmt.Errorf("failed to create role %s: %w", name, err)
		}
		if name == adminRoleName {
			adminRoleID = roleID
		}
	}
	if adminRoleID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("admin role %s not found", adminRoleName)
	}
	user.TenantID = tenantID
	userID, err := r.tenants.CreateUser(ctx, tx, user)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create admin user: %w", err)
	}
	if err := r.tenants.AttachRole(ctx, tx, tenantID, userID, adminRoleID); err != nil {
		return uuid.Nil, fmt.Errorf("failed to attach admin role: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}
