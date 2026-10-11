package campaign

import (
	"encoding/json"

	"github.com/google/uuid"
)

type CreateCampaignRequest struct {
	Name            string          `json:"name" validate:"required"`
	Type            string          `json:"type"`
	Description     *string         `json:"description,omitempty"`
	MessageTemplate string          `json:"message_template" validate:"required"`
	ChannelID       uuid.UUID       `json:"channel_id" validate:"required,uuid"`
	SendingOption   json.RawMessage `json:"sending_option,omitempty"`
	ScheduledAt     *string         `json:"scheduled_at,omitempty"`
	RecipientListID *uuid.UUID      `json:"recipient_list_id,omitempty" validate:"omitnil,required,uuid"`
	TemplateID      *uuid.UUID      `json:"template_id,omitempty" validate:"omitnil,required,uuid"`
}

type UpdateCampaignRequest struct {
	Name            *string         `json:"name,omitempty"`
	Type            *string         `json:"type,omitempty"`
	Description     *string         `json:"description,omitempty"`
	MessageTemplate *string         `json:"message_template,omitempty"`
	Status          *string         `json:"status,omitempty"`
	SendingOption   json.RawMessage `json:"sending_option,omitempty"`
	ScheduledAt     *string         `json:"scheduled_at,omitempty"`
	RecipientListID *uuid.UUID      `json:"recipient_list_id,omitempty" validate:"omitnil,required,uuid"`
	TemplateID      *uuid.UUID      `json:"template_id,omitempty" validate:"omitnil,required,uuid"`
	AgentID         *uuid.UUID      `json:"agent_id,omitempty" validate:"omitnil,required,uuid"`
	SenderID        *uuid.UUID      `json:"sender_id,omitempty" validate:"omitnil,required,uuid"`
}

type ListCampaignQuery struct {
	Page   int    `json:"page"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

type CreateTemplateRequest struct {
	Name         string          `json:"name" validate:"required"`
	Type         *string         `json:"type,omitempty"`
	TemplateType *string         `json:"template_type,omitempty"`
	Category     *string         `json:"category,omitempty"`
	Language     *string         `json:"language,omitempty"`
	Content      json.RawMessage `json:"content" validate:"required"`
	Variables    json.RawMessage `json:"variables"`
	Quality      *string         `json:"quality,omitempty"`
	AccountID    *uuid.UUID      `json:"account_id,omitempty" validate:"omitnil,required,uuid"`
}

type UpdateTemplateRequest struct {
	Name         *string         `json:"name,omitempty"`
	Type         *string         `json:"type,omitempty"`
	TemplateType *string         `json:"template_type,omitempty"`
	Category     *string         `json:"category,omitempty"`
	Language     *string         `json:"language,omitempty"`
	Content      json.RawMessage `json:"content,omitempty"`
	Variables    json.RawMessage `json:"variables,omitempty"`
	Quality      *string         `json:"quality,omitempty"`
	AccountID    *uuid.UUID      `json:"account_id,omitempty" validate:"omitnil,required,uuid"`
}

type CreateRecipientListRequest struct {
	Name      string     `json:"name" validate:"required"`
	Source    string     `json:"source"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty" validate:"omitnil,required,uuid"`
}

type UpdateRecipientListRequest struct {
	Name      *string    `json:"name,omitempty"`
	Source    *string    `json:"source,omitempty"`
	Status    *string    `json:"status,omitempty"`
	ChannelID *uuid.UUID `json:"channel_id,omitempty" validate:"omitnil,required,uuid"`
}

type AddContactToListRequest struct {
	FirstName       string     `json:"first_name" validate:"required"`
	LastName        *string    `json:"last_name,omitempty"`
	Username        *string    `json:"username,omitempty"`
	Institution     *string    `json:"institution,omitempty"`
	Email           *string    `json:"email,omitempty"`
	Phone           *string    `json:"phone,omitempty"`
	MasterContactID *uuid.UUID `json:"master_contact_id,omitempty" validate:"omitnil,required,uuid"`
}
