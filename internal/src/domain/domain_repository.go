package domain

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

type DomainRepository interface {
	List(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Domain, error)
	GetByTenantIDs(ctx context.Context, q DBTX, tenantIDs []uuid.UUID) (map[uuid.UUID][]Domain, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Domain, error)
	Create(ctx context.Context, q DBTX, d *Domain) (uuid.UUID, error)
	Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, d *Domain) error
	Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
}
