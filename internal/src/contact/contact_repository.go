package contact

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type ContactRepository interface {
	List(ctx context.Context, q DBTX, tenantID int, limit, offset int, f ListContactQuery) ([]*Contact, int, error)
	GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Contact, error)
	GetByPhone(ctx context.Context, q DBTX, tenantID int, phone string) (*Contact, error)
	Create(ctx context.Context, q DBTX, contact *Contact) (int, error)
	Update(ctx context.Context, q DBTX, tenantID int, id int, contact *Contact) error
	SoftDelete(ctx context.Context, q DBTX, tenantID int, id int) error
}
