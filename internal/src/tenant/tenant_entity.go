package tenant

import (
	"encoding/json"
	"time"
)

type Tenant struct {
	ID        int             `json:"id"`
	Name      string          `json:"name"`
	Domain    string          `json:"domain"`
	Logo      json.RawMessage `json:"logo,omitempty"`
	Address   *string         `json:"address,omitempty"`
	Phone     *string         `json:"phone,omitempty"`
	Email     *string         `json:"email,omitempty"`
	IsActive  bool            `json:"is_active"`
	Settings  json.RawMessage `json:"settings,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
