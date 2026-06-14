package campaign

import (
	"encoding/json"
	"time"
)

type Campaign struct {
	ID              int              `json:"id"`
	TenantID        int              `json:"tenant_id"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	Description     *string          `json:"description,omitempty"`
	MessageTemplate string           `json:"message_template"`
	ChannelID       int              `json:"channel_id"`
	Status          string           `json:"status"`
	SendingOption   json.RawMessage  `json:"sending_option,omitempty"`
	Stats           json.RawMessage  `json:"stats,omitempty"`
	ScheduledAt     *time.Time       `json:"scheduled_at,omitempty"`
	SentCount       int              `json:"sent_count"`
	TotalCount      int              `json:"total_count"`
	AgentID         *int             `json:"agent_id,omitempty"`
	SenderID        *int             `json:"sender_id,omitempty"`
	RecipientListID *int             `json:"recipient_list_id,omitempty"`
	TemplateID      *int             `json:"template_id,omitempty"`
	CreatedBy       int              `json:"created_by"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

type CampaignTemplate struct {
	ID           int              `json:"id"`
	TenantID     int              `json:"tenant_id"`
	Name         string           `json:"name"`
	Type         *string          `json:"type,omitempty"`
	TemplateType *string          `json:"template_type,omitempty"`
	Category     *string          `json:"category,omitempty"`
	Language     *string          `json:"language,omitempty"`
	Content      json.RawMessage  `json:"content"`
	Variables    json.RawMessage  `json:"variables"`
	Quality      *string          `json:"quality,omitempty"`
	AccountID    *int             `json:"account_id,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

type CampaignRecipientList struct {
	ID        int        `json:"id"`
	TenantID  int        `json:"tenant_id"`
	Name      string     `json:"name"`
	Source    string     `json:"source"`
	Status    *string    `json:"status,omitempty"`
	ChannelID *int       `json:"channel_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CampaignRecipientContact struct {
	ID                    int        `json:"id"`
	FirstName             string     `json:"first_name"`
	LastName              *string    `json:"last_name,omitempty"`
	Username              *string    `json:"username,omitempty"`
	Institution           *string    `json:"institution,omitempty"`
	Email                 *string    `json:"email,omitempty"`
	Phone                 *string    `json:"phone,omitempty"`
	CampaignRecipientListID int      `json:"campaign_recipient_list_id"`
	MasterContactID       *int       `json:"master_contact_id,omitempty"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`
}

type CampaignRecipient struct {
	ID                int        `json:"id"`
	CampaignID        int        `json:"campaign_id"`
	RecipientContactID int       `json:"recipient_contact_id"`
	Status            string     `json:"status"`
	FailedReason      *string    `json:"failed_reason,omitempty"`
	DeliveryTime      *time.Time `json:"delivery_time,omitempty"`
	OpenTime          *time.Time `json:"open_time,omitempty"`
	ClickTime         *time.Time `json:"click_time,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type PaginationMeta struct {
	Total       int `json:"total"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	From        int `json:"from"`
	To          int `json:"to"`
}

type PaginatedResponse struct {
	Meta PaginationMeta `json:"meta"`
	Data interface{}    `json:"data"`
}
