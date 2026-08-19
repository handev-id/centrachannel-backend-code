package message

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type MessageRepository interface {
	ListCursor(ctx context.Context, q DBTX, tenantID int, conversationID int, limit int, lastID int) ([]*Message, error)
	Create(ctx context.Context, q DBTX, msg *Message) (int, error)
	UpdateStatus(ctx context.Context, q DBTX, id int, status string) error
	UpdateStatusByWebhookID(ctx context.Context, q DBTX, webhookMessageID string, status string) error
	UpdateWebhookID(ctx context.Context, q DBTX, id int, webhookMessageID string) error
}
