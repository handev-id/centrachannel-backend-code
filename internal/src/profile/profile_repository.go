package profile

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type ProfileRepository interface {
	List(ctx context.Context, q DBTX, contactID, channelID int) ([]Profile, error)
	GetByID(ctx context.Context, q DBTX, id int) (*Profile, error)
	GetByExternalIDAndChannelID(ctx context.Context, q DBTX, externalID string, channelID int) (*Profile, error)
	Create(ctx context.Context, q DBTX, profile *Profile) (int, error)
	Update(ctx context.Context, q DBTX, id int, profile *Profile) error
	GetByContactID(ctx context.Context, q DBTX, contactID int) ([]Profile, error)
	GetByContactIDs(ctx context.Context, q DBTX, contactIDs []int) (map[int][]Profile, error)
}
