package message

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

type MessageRepository interface {
	ListCursor(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error)
	Create(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error)
	UpdateStatus(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error
	UpdateStatusByWebhookID(ctx context.Context, q DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error
	UpdateWebhookID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, webhookMessageID string) error
	MarkReadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) error
	GetUnreadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]*Message, error)
}
