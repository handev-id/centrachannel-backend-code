package registration

import "centrachannel/internal/src/tenant"

type OnboardRequest struct {
	Name          string `json:"name" validate:"required"`
	Domain        string `json:"domain" validate:"required"`
	AdminEmail    string `json:"admin_email" validate:"required,email"`
	AdminPassword string `json:"admin_password" validate:"required,min=8"`
	CompanyName   string `json:"company_name"`
}

type OnboardResponse struct {
	Tenant *tenant.Tenant `json:"tenant"`
	Admin  *AdminUser     `json:"admin"`
}

type AdminUser struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	Email     string `json:"email"`
	Username  string `json:"username"`
}
