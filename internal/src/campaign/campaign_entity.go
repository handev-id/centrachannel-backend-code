package campaign

import (
	"time"
)

type Campaign struct {
	ID              int              `json:"id"`
	TenantID        int              `json:"tenant_id"`
	Name            string           `json:"name"`
	Description     *string          `json:"description,omitempty"`
	MessageTemplate string           `json:"message_template"`
	ChannelID       int              `json:"channel_id"`
	Status          string           `json:"status"`
	ScheduledAt     *time.Time       `json:"scheduled_at,omitempty"`
	SentCount       int              `json:"sent_count"`
	TotalCount      int              `json:"total_count"`
	CreatedBy       int              `json:"created_by"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CampaignContact struct {
	ID           int        `json:"id"`
	CampaignID   int        `json:"campaign_id"`
	ContactID    int        `json:"contact_id"`
	Status       string     `json:"status"`
	SentAt       *time.Time `json:"sent_at,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}
