package conversation

import (
	"context"
	"database/sql"
	"time"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type ConversationRepository interface {
	List(ctx context.Context, q DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*Conversation, int, error)
	ListCursor(ctx context.Context, q DBTX, tenantID int, limit int, status string, channelID, agentID int, search string, lastActivity *time.Time, lastID int) ([]*Conversation, error)
	GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Conversation, error)
	FindOpenByProfileAndChannel(ctx context.Context, q DBTX, tenantID int, profileID int, channelID int) (*Conversation, error)
	Create(ctx context.Context, q DBTX, conv *Conversation) (int, error)
	UpdateStatus(ctx context.Context, q DBTX, tenantID int, id int, status string) error
	Assign(ctx context.Context, q DBTX, tenantID int, id int, agentID int) error
	Unassign(ctx context.Context, q DBTX, tenantID int, id int) error
	MarkRead(ctx context.Context, q DBTX, tenantID int, id int) error
	UpdateLastMessage(ctx context.Context, q DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error
	GetTotalUnread(ctx context.Context, q DBTX, tenantID int) (int, error)
}
