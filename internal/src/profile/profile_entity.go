package profile

import (
	"time"

	"github.com/google/uuid"

	"centrachannel/internal/utils"
)

type Profile struct {
	TenantID               uuid.UUID          `json:"tenant_id"`
	ID                     uuid.UUID          `json:"id"`
	ExternalID             string             `json:"external_id"`
	Username               *string            `json:"username,omitempty"`
	DisplayName            *string            `json:"display_name,omitempty"`
	IsMain                 bool               `json:"is_main"`
	LinkedDeviceWhatsappID *string            `json:"linked_device_whatsapp_id,omitempty"`
	MergedFromContactID    *uuid.UUID         `json:"merged_from_contact_id,omitempty"`
	ContactID              uuid.UUID          `json:"contact_id"`
	ChannelID              uuid.UUID          `json:"channel_id"`
	DeletedAt              utils.NullableTime `json:"deleted_at,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}
