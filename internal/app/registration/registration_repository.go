package registration

import (
	"context"
	"database/sql"

	"github.com/google/uuid"

	"centrachannel/internal/src/domain"
	"centrachannel/internal/src/tenant"
)

type RegistrationRepository interface {
	Create(ctx context.Context, db *sql.DB, t *tenant.Tenant, d *domain.Domain, user *tenant.User, roleNames []string, adminRoleName string) (uuid.UUID, error)
}
