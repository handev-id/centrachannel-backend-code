package campaign

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Campaign struct {
	ID              uuid.UUID       `json:"id"`
	TenantID        uuid.UUID       `json:"tenant_id"`
	Name            string          `json:"name"`
	Type            string          `json:"type"`
	Description     *string         `json:"description,omitempty"`
	MessageTemplate string          `json:"message_template"`
	ChannelID       uuid.UUID       `json:"channel_id"`
	Status          string          `json:"status"`
	SendingOption   json.RawMessage `json:"sending_option,omitempty"`
	Stats           json.RawMessage `json:"stats,omitempty"`
	ScheduledAt     *time.Time      `json:"scheduled_at,omitempty"`
	SentCount       int             `json:"sent_count"`
	TotalCount      int             `json:"total_count"`
	AgentID         *uuid.UUID      `json:"agent_id,omitempty"`
	SenderID        *uuid.UUID      `json:"sender_id,omitempty"`
	RecipientListID *uuid.UUID      `json:"recipient_list_id,omitempty"`
	TemplateID      *uuid.UUID      `json:"template_id,omitempty"`
	CreatedBy       uuid.UUID       `json:"created_by"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CampaignTemplate struct {
	ID           uuid.UUID       `json:"id"`
	TenantID     uuid.UUID       `json:"tenant_id"`
	Name         string          `json:"name"`
	Type         *string         `json:"type,omitempty"`
	TemplateType *string         `json:"template_type,omitempty"`
	Category     *string         `json:"category,omitempty"`
	Language     *string         `json:"language,omitempty"`
	Content      json.RawMessage `json:"content"`
	Variables    json.RawMessage `json:"variables"`
	Quality      *string         `json:"quality,omitempty"`
	AccountID    *uuid.UUID      `json:"account_id,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type CampaignRecipientList struct {
	ID        uuid.UUID  `json:"id"`
	TenantID  uuid.UUID  `json:"tenant_id"`
	Name      string     `json:"name"`
	Source    string     `json:"source"`
	Status    *string    `json:"status,omitempty"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type CampaignRecipientContact struct {
	ID                      uuid.UUID  `json:"id"`
	FirstName               string     `json:"first_name"`
	LastName                *string    `json:"last_name,omitempty"`
	Username                *string    `json:"username,omitempty"`
	Institution             *string    `json:"institution,omitempty"`
	Email                   *string    `json:"email,omitempty"`
	Phone                   *string    `json:"phone,omitempty"`
	CampaignRecipientListID uuid.UUID  `json:"campaign_recipient_list_id"`
	MasterContactID         *uuid.UUID `json:"master_contact_id,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at"`
}

type CampaignRecipientWithPhone struct {
	CampaignRecipient
	Phone     string `json:"phone"`
	Message   string `json:"message,omitempty"`
	FirstName string `json:"first_name,omitempty"`
}

type CampaignRecipient struct {
	ID                 uuid.UUID  `json:"id"`
	CampaignID         uuid.UUID  `json:"campaign_id"`
	RecipientContactID uuid.UUID  `json:"recipient_contact_id"`
	Status             string     `json:"status"`
	FailedReason       *string    `json:"failed_reason,omitempty"`
	DeliveryTime       *time.Time `json:"delivery_time,omitempty"`
	OpenTime           *time.Time `json:"open_time,omitempty"`
	ClickTime          *time.Time `json:"click_time,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}
