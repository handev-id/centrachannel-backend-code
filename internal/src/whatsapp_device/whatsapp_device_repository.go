package whatsapp_device

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

type WhatsAppDeviceRepository interface {
	List(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]WhatsAppDevice, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*WhatsAppDevice, error)
	GetByWhatsappID(ctx context.Context, q DBTX, whatsappID string) (*WhatsAppDevice, error)
	Create(ctx context.Context, q DBTX, device *WhatsAppDevice) (uuid.UUID, error)
	Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, device *WhatsAppDevice) error
	Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
}
