package conversation

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type ConversationRepository interface {
	ListCursor(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, status string, channelID uuid.UUID, agentID uuid.UUID, search string, lastActivity *time.Time, lastID uuid.UUID) ([]*Conversation, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Conversation, error)
	FindOpenByProfileAndChannel(ctx context.Context, q DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*Conversation, error)
	Create(ctx context.Context, q DBTX, conv *Conversation) (uuid.UUID, error)
	UpdateStatus(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error
	Assign(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, agentID uuid.UUID) error
	Unassign(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	MarkRead(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	UpdateLastMessage(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error
	GetTotalUnread(ctx context.Context, q DBTX, tenantID uuid.UUID) (int, error)
}
