package channel

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Channel struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Type      string          `json:"type"`
	Logo      json.RawMessage `json:"logo,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}
