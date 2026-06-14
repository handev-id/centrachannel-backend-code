package tag

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type TagRepository interface {
	List(ctx context.Context, q DBTX, tenantID int) ([]Tag, error)
	GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Tag, error)
	Create(ctx context.Context, q DBTX, tag *Tag) (int, error)
	Update(ctx context.Context, q DBTX, tenantID int, id int, tag *Tag) error
	Delete(ctx context.Context, q DBTX, tenantID int, id int) error
}
