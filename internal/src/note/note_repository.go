package note

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type NoteRepository interface {
	ListByConversation(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]Note, error)
	Create(ctx context.Context, q DBTX, note *Note) (int, error)
	Update(ctx context.Context, q DBTX, tenantID int, id int, note *Note) error
	Delete(ctx context.Context, q DBTX, tenantID int, id int) error
}
