package user

import (
	"encoding/json"
	"time"

	"centrachannel/internal/utils"
)

type Avatar struct {
	Name    string  `json:"name"`
	Extname string  `json:"extname"`
	Size    int64   `json:"size"`
	Type    string  `json:"type"`
	URL     *string `json:"url,omitempty"`
}

type User struct {
	ID        int             `json:"id"`
	TenantID  int             `json:"tenant_id"`
	FirstName string          `json:"first_name"`
	LastName  *string         `json:"last_name,omitempty"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Phone     *string         `json:"phone,omitempty"`
	Password  string          `json:"-"`
	Avatar    json.RawMessage `json:"avatar,omitempty"`
	LastLogin *time.Time      `json:"last_login,omitempty"`
	DeletedAt utils.NullableTime `json:"deleted_at,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Roles     []Role          `json:"roles,omitempty"`
}

type Role struct {
	ID        int       `json:"id"`
	TenantID  int       `json:"tenant_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
