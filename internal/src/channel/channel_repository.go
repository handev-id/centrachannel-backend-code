package channel

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type ChannelRepository interface {
	List(ctx context.Context, q DBTX) ([]Channel, error)
	GetByID(ctx context.Context, q DBTX, id int) (*Channel, error)
	GetByType(ctx context.Context, q DBTX, channelType string) (*Channel, error)
}
