package conversation_tag

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

type ConversationTagRepository interface {
	ListByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]ConversationTag, error)
	Attach(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, tagID uuid.UUID) error
	Detach(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, tagID uuid.UUID) error
}
