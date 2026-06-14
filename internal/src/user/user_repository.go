package user

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type UserRepository interface {
	List(ctx context.Context, q DBTX, tenantID int, limit, offset int, search string, roleID *int) ([]*User, int, error)
	GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*User, error)
	Create(ctx context.Context, q DBTX, user *User) (int, error)
	Update(ctx context.Context, q DBTX, tenantID int, id int, user *User) error
	SoftDelete(ctx context.Context, q DBTX, tenantID int, id int) error
	GetRolesByUserID(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error)
	GetRolesByUserIDs(ctx context.Context, q DBTX, tenantID int, userIDs []int) (map[int][]Role, error)
	AttachRoles(ctx context.Context, q DBTX, tenantID int, userID int, roleIDs []int) error
	SyncRoles(ctx context.Context, q DBTX, tenantID int, userID int, roleIDs []int) error
}
