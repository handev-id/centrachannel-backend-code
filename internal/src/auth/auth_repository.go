package auth

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type AuthRepository interface {
	GetByEmail(ctx context.Context, q DBTX, tenantID uuid.UUID, email string) (*User, error)
	GetByUsername(ctx context.Context, q DBTX, tenantID uuid.UUID, username string) (*User, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*User, error)
	Create(ctx context.Context, q DBTX, user *User) (uuid.UUID, error)
	GetRolesByUserID(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID) ([]Role, error)
	GetRoleIDByName(ctx context.Context, q DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error)
	AttachRole(ctx context.Context, q DBTX, tenantID, userID, roleID uuid.UUID) error
	UpdateLastLogin(ctx context.Context, q DBTX, tenantID, userID uuid.UUID) error
}
