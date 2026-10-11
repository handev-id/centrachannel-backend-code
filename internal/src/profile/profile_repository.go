package profile

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

type ProfileRepository interface {
	List(ctx context.Context, q DBTX, tenantID uuid.UUID, contactID uuid.UUID, channelID uuid.UUID) ([]Profile, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Profile, error)
	GetByExternalIDAndChannelID(ctx context.Context, q DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*Profile, error)
	Create(ctx context.Context, q DBTX, profile *Profile) (uuid.UUID, error)
	Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, profile *Profile) error
	GetByContactID(ctx context.Context, q DBTX, tenantID uuid.UUID, contactID uuid.UUID) ([]Profile, error)
	GetByContactIDs(ctx context.Context, q DBTX, tenantID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID][]Profile, error)
}
