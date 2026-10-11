package registration

import (
	"errors"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/src/domain"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/hostname"
	"centrachannel/internal/utils/response"
)

type RegistrationHandler struct {
	service RegistrationService
}

func NewRegistrationHandler(c *di.Container) *RegistrationHandler {
	repo := tenant.NewTenantRepository()
	service := NewRegistrationService(repo, domain.NewDomainRepository(), c.DB, c.Config, c.Logger)
	return &RegistrationHandler{service: service}
}

func (h *RegistrationHandler) Onboard(c fiber.Ctx) error {
	var req OnboardRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(&req); err != nil {
		return response.ValidationFailed(c, err)
	}

	result, err := h.service.Onboard(c.Context(), req)
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			return response.Error(c, fiber.StatusConflict, domain.ErrConflict.Error(), nil)
		}
		if errors.Is(err, hostname.ErrInvalidName) {
			return response.UnprocessableEntity(c, err.Error(), nil)
		}
		return response.InternalServerError(c, "Failed to onboard tenant")
	}

	return response.Created(c, "Tenant onboarded successfully", result)
}
