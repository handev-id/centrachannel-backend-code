package registration

import (
	"github.com/google/uuid"

	"centrachannel/internal/src/tenant"
)

type OnboardRequest struct {
	Name          string `json:"name" validate:"required"`
	Domain        string `json:"domain" validate:"required,max=254"`
	AdminEmail    string `json:"admin_email" validate:"required,email"`
	AdminPassword string `json:"admin_password" validate:"required,min=8"`
	CompanyName   string `json:"company_name"`
}

type OnboardResponse struct {
	Tenant *tenant.TenantResponse `json:"tenant"`
	Admin  *AdminUser             `json:"admin"`
}

type AdminUser struct {
	ID        uuid.UUID `json:"id"`
	FirstName string    `json:"first_name"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
}
