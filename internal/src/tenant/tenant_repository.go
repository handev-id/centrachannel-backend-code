package tenant

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type User struct {
	FirstName string
	LastName  *string
	Username  string
	Email     string
	Password  string
	TenantID  uuid.UUID
}

type TenantRepository interface {
	GetDomains(ctx context.Context, q DBTX, tenantID uuid.UUID) (json.RawMessage, error)
	GetActiveIDByDomain(ctx context.Context, q DBTX, name string) (uuid.UUID, error)
	Create(ctx context.Context, q DBTX, tenant *Tenant) (uuid.UUID, error)
	CreateRole(ctx context.Context, q DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error)
	CreateUser(ctx context.Context, q DBTX, user *User) (uuid.UUID, error)
	AttachRole(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID, roleID uuid.UUID) error
	GetByID(ctx context.Context, q DBTX, id uuid.UUID) (*Tenant, error)
	Update(ctx context.Context, q DBTX, tenant *Tenant) error
	GetByMetaPageID(ctx context.Context, q DBTX, pageID string) (*Tenant, error)
	GetByMetaInstagramBusinessID(ctx context.Context, q DBTX, igID string) (*Tenant, error)
}
