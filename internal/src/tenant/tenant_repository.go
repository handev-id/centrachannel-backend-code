package tenant

import (
	"context"
	"database/sql"
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
	TenantID  int
}

type TenantRepository interface {
	Create(ctx context.Context, q DBTX, tenant *Tenant) (int, error)
	CreateRole(ctx context.Context, q DBTX, tenantID int, name string) (int, error)
	CreateUser(ctx context.Context, q DBTX, user *User) (int, error)
	AttachRole(ctx context.Context, q DBTX, tenantID, userID, roleID int) error
	GetByID(ctx context.Context, q DBTX, id int) (*Tenant, error)
	Update(ctx context.Context, q DBTX, tenant *Tenant) error
	GetByMetaPageID(ctx context.Context, q DBTX, pageID string) (*Tenant, error)
	GetByMetaInstagramBusinessID(ctx context.Context, q DBTX, igID string) (*Tenant, error)
}
