package channel

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

type ChannelRepository interface {
	List(ctx context.Context, q DBTX) ([]Channel, error)
	GetByID(ctx context.Context, q DBTX, id uuid.UUID) (*Channel, error)
	GetByType(ctx context.Context, q DBTX, channelType string) (*Channel, error)
}
