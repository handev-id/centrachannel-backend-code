package registration

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

type RegistrationHandler struct {
	service RegistrationService
}

func NewRegistrationHandler(c *di.Container) *RegistrationHandler {
	repo := tenant.NewTenantRepository()
	service := NewRegistrationService(repo, c.DB, c.Config, c.Logger)
	return &RegistrationHandler{service: service}
}

func (h *RegistrationHandler) Onboard(c fiber.Ctx) error {
	var req OnboardRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	result, err := h.service.Onboard(c.Context(), req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Created(c, "Tenant onboarded successfully", result)
}
