package contact

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

type ContactRepository interface {
	List(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, f ListContactQuery) ([]*Contact, int, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Contact, error)
	GetByPhone(ctx context.Context, q DBTX, tenantID uuid.UUID, phone string) (*Contact, error)
	Create(ctx context.Context, q DBTX, contact *Contact) (uuid.UUID, error)
	Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, contact *Contact) error
	SoftDelete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	ListConversations(ctx context.Context, q DBTX, tenantID, contactID uuid.UUID, limit, offset int) ([]ConversationBrief, int, error)
	Merge(ctx context.Context, q DBTX, tenantID, sourceID, targetID uuid.UUID) error
	Unmerge(ctx context.Context, q DBTX, tenantID, id uuid.UUID) error
}
