package profile

import (
	"database/sql"
	"time"
)

type Profile struct {
	ID                      int          `json:"id"`
	ExternalID              string       `json:"external_id"`
	Username                *string      `json:"username,omitempty"`
	DisplayName             *string      `json:"display_name,omitempty"`
	IsMain                  bool         `json:"is_main"`
	LinkedDeviceWhatsappID  *string      `json:"linked_device_whatsapp_id,omitempty"`
	MergedFromContactID     *int         `json:"merged_from_contact_id,omitempty"`
	ContactID               int          `json:"contact_id"`
	ChannelID               int          `json:"channel_id"`
	DeletedAt               sql.NullTime `json:"deleted_at,omitempty"`
	CreatedAt               time.Time    `json:"created_at"`
	UpdatedAt               time.Time    `json:"updated_at"`
}
