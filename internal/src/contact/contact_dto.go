package contact

import "encoding/json"

type ListContactQuery struct {
	Page             int     `query:"page"`
	Limit            int     `query:"limit"`
	Search           string  `query:"search"`
	Status           string  `query:"status"`
	ChannelID        int     `query:"channel_id"`
	ChannelType      string  `query:"channel_type"`
	Category         string  `query:"category"`
	Country          string  `query:"country"`
	Province         string  `query:"province"`
	AgentID          int     `query:"agent_id"`
	IsMerged         string  `query:"is_merged"`
	HasConversation  string  `query:"has_conversation"`
	LastActivityFrom string  `query:"last_activity_from"`
	LastActivityTo   string  `query:"last_activity_to"`
	SortBy           string  `query:"sort_by"`
}

type CreateContactRequest struct {
	FirstName           string          `json:"first_name" validate:"required,max=255"`
	LastName            *string         `json:"last_name,omitempty"`
	Username            *string         `json:"username,omitempty"`
	Email               *string         `json:"email,omitempty"`
	Phone               *string         `json:"phone,omitempty"`
	Avatar              json.RawMessage `json:"avatar,omitempty"`
	Country             *string         `json:"country,omitempty"`
	Bio                 *string         `json:"bio,omitempty"`
	Occupation          *string         `json:"occupation,omitempty"`
	Category            *string         `json:"category,omitempty"`
	CategoryDescription *string         `json:"category_description,omitempty"`
	Gender              *string         `json:"gender,omitempty"`
	DateOfBirth         *string         `json:"date_of_birth,omitempty"`
	ProvinceOfOrigin    *string         `json:"province_of_origin,omitempty"`
	Facebook            *string         `json:"facebook,omitempty"`
	Instagram           *string         `json:"instagram,omitempty"`
	Whatsapp            *string         `json:"whatsapp,omitempty"`
	X                   *string         `json:"x,omitempty"`
	Tiktok              *string         `json:"tiktok,omitempty"`
	Status              *string         `json:"status,omitempty"`
	InstitutionName     *string         `json:"institution_name,omitempty"`
}

type UpdateContactRequest struct {
	FirstName           *string         `json:"first_name,omitempty"`
	LastName            *string         `json:"last_name,omitempty"`
	Username            *string         `json:"username,omitempty"`
	Email               *string         `json:"email,omitempty"`
	Phone               *string         `json:"phone,omitempty"`
	Avatar              json.RawMessage `json:"avatar,omitempty"`
	Country             *string         `json:"country,omitempty"`
	Bio                 *string         `json:"bio,omitempty"`
	Occupation          *string         `json:"occupation,omitempty"`
	Category            *string         `json:"category,omitempty"`
	CategoryDescription *string         `json:"category_description,omitempty"`
	Gender              *string         `json:"gender,omitempty"`
	DateOfBirth         *string         `json:"date_of_birth,omitempty"`
	ProvinceOfOrigin    *string         `json:"province_of_origin,omitempty"`
	Facebook            *string         `json:"facebook,omitempty"`
	Instagram           *string         `json:"instagram,omitempty"`
	Whatsapp            *string         `json:"whatsapp,omitempty"`
	X                   *string         `json:"x,omitempty"`
	Tiktok              *string         `json:"tiktok,omitempty"`
	Status              *string         `json:"status,omitempty"`
	InstitutionName     *string         `json:"institution_name,omitempty"`
}

type CSVImportResult struct {
	Total   int              `json:"total"`
	Success int              `json:"success"`
	Failed  int              `json:"failed"`
	Errors  []CSVImportError `json:"errors,omitempty"`
}

type CSVImportError struct {
	Row   int    `json:"row"`
	Field string `json:"field"`
	Error string `json:"error"`
}

type MergeContactRequest struct {
	TargetContactID int `json:"target_contact_id" validate:"required"`
}

type PaginatedResponse struct {
	Meta PaginationMeta `json:"meta"`
	Data interface{}    `json:"data"`
}

type PaginationMeta struct {
	Total       int `json:"total"`
	PerPage     int `json:"per_page"`
	CurrentPage int `json:"current_page"`
	LastPage    int `json:"last_page"`
	From        int `json:"from"`
	To          int `json:"to"`
}
