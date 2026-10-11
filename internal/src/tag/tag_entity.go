package tag

import (
	"time"

	"github.com/google/uuid"
)

type Tag struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Name      string    `json:"name"`
	Color     *string   `json:"color,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
