package user

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

type UserRepository interface {
	ListRoles(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Role, error)
	List(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, search string, roleID *uuid.UUID) ([]*User, int, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*User, error)
	Create(ctx context.Context, db *sql.DB, user *User, roleIDs []uuid.UUID) (uuid.UUID, error)
	Update(ctx context.Context, db *sql.DB, tenantID, id uuid.UUID, user *User, roleIDs []uuid.UUID) error
	SoftDelete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	GetRolesByUserID(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID) ([]Role, error)
	GetRolesByUserIDs(ctx context.Context, q DBTX, tenantID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID][]Role, error)
	AttachRoles(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID, roleIDs []uuid.UUID) error
	SyncRoles(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID, roleIDs []uuid.UUID) error
}
