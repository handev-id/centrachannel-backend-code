package whatsapp_device

import (
	"database/sql"
	"time"
)

type WhatsAppDevice struct {
	ID          int          `json:"id"`
	TenantID    int          `json:"tenant_id"`
	Name        string       `json:"name"`
	CountryCode string       `json:"country_code"`
	Phone       string       `json:"phone"`
	WhatsappID  string       `json:"whatsapp_id"`
	Status      string       `json:"status"`
	DeletedAt   sql.NullTime `json:"deleted_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}
