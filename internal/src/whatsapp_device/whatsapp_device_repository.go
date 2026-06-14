package whatsapp_device

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type WhatsAppDeviceRepository interface {
	List(ctx context.Context, q DBTX, tenantID int) ([]WhatsAppDevice, error)
	GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error)
	GetByWhatsappID(ctx context.Context, q DBTX, whatsappID string) (*WhatsAppDevice, error)
	Create(ctx context.Context, q DBTX, device *WhatsAppDevice) (int, error)
	Update(ctx context.Context, q DBTX, tenantID int, id int, device *WhatsAppDevice) error
	Delete(ctx context.Context, q DBTX, tenantID int, id int) error
}
