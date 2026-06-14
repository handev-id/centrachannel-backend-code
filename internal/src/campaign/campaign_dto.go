package campaign

type CreateCampaignRequest struct {
	Name            string  `json:"name" validate:"required"`
	Description     *string `json:"description,omitempty"`
	MessageTemplate string  `json:"message_template" validate:"required"`
	ChannelID       int     `json:"channel_id" validate:"required"`
	ScheduledAt     *string `json:"scheduled_at,omitempty"`
	ContactIDs      []int   `json:"contact_ids" validate:"required,min=1"`
}

type UpdateCampaignRequest struct {
	Name            *string `json:"name,omitempty"`
	Description     *string `json:"description,omitempty"`
	MessageTemplate *string `json:"message_template,omitempty"`
	Status          *string `json:"status,omitempty"`
	ScheduledAt     *string `json:"scheduled_at,omitempty"`
}
