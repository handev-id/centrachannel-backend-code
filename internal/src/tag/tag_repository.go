package tag

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

type TagRepository interface {
	List(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Tag, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Tag, error)
	Create(ctx context.Context, q DBTX, tag *Tag) (uuid.UUID, error)
	Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, tag *Tag) error
	Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
}
