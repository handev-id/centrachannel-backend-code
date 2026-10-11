package note

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

type NoteRepository interface {
	ListByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]Note, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Note, error)
	Create(ctx context.Context, q DBTX, note *Note) (uuid.UUID, error)
	Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, note *Note) error
	Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
}
