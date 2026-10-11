package domain

type CreateDomainRequest struct {
	Name      string `json:"name" validate:"required,max=254"`
	IsDefault *bool  `json:"is_default,omitempty"`
	IsActive  *bool  `json:"is_active,omitempty"`
}

type UpdateDomainRequest struct {
	Name      *string `json:"name,omitempty" validate:"omitempty,max=254"`
	IsDefault *bool   `json:"is_default,omitempty"`
	IsActive  *bool   `json:"is_active,omitempty"`
}
