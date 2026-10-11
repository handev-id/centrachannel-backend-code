package tenant

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Tenant struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Logo      json.RawMessage `json:"logo,omitempty"`
	Address   *string         `json:"address,omitempty"`
	Phone     *string         `json:"phone,omitempty"`
	Email     *string         `json:"email,omitempty"`
	IsActive  bool            `json:"is_active"`
	Settings  json.RawMessage `json:"settings,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
