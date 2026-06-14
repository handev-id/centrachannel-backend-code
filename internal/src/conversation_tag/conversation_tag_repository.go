package conversation_tag

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type ConversationTagRepository interface {
	ListByConversation(ctx context.Context, q DBTX, tenantID int, conversationID int) ([]ConversationTag, error)
	Attach(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error
	Detach(ctx context.Context, q DBTX, tenantID int, conversationID int, tagID int) error
}
