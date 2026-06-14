package auth

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type AuthRepository interface {
	GetByEmail(ctx context.Context, q DBTX, tenantID int, email string) (*User, error)
	GetByUsername(ctx context.Context, q DBTX, tenantID int, username string) (*User, error)
	GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*User, error)
	Create(ctx context.Context, q DBTX, user *User) (int, error)
	GetRolesByUserID(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error)
}
