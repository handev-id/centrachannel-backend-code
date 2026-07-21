package tenant

import "encoding/json"

type UpdateTenantRequest struct {
	Name     *string          `json:"name,omitempty" validate:"omitempty,max=255"`
	Domain   *string          `json:"domain,omitempty" validate:"omitempty,max=255"`
	Logo     json.RawMessage  `json:"logo,omitempty"`
	Address  *string          `json:"address,omitempty" validate:"omitempty,max=255"`
	Phone    *string          `json:"phone,omitempty" validate:"omitempty,max=50"`
	Email    *string          `json:"email,omitempty" validate:"omitempty,email"`
	IsActive *bool            `json:"is_active,omitempty"`
	Settings json.RawMessage  `json:"settings,omitempty"`
}
