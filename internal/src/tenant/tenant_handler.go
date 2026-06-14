package tenant

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/utils/response"
)

type TenantHandler struct {
	service TenantService
}

func NewTenantHandler(c *di.Container) *TenantHandler {
	repo := NewTenantRepository()
	service := NewTenantService(repo, c.DB, c.Config, c.Logger)
	return &TenantHandler{service: service}
}

func (h *TenantHandler) Onboard(c fiber.Ctx) error {
	var req OnboardRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	result, err := h.service.Onboard(c.Context(), req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}

	return response.Created(c, "Tenant onboarded successfully", result)
}

func (h *TenantHandler) List(c fiber.Ctx) error {
	tenants, err := h.service.List(c.Context())
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Tenants retrieved successfully", tenants)
}

func (h *TenantHandler) Show(c fiber.Ctx) error {
	idStr := c.Params("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return response.BadRequest(c, "Invalid tenant ID", nil)
	}

	tenant, err := h.service.GetByID(c.Context(), id)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	if tenant == nil {
		return response.NotFound(c, "Tenant not found")
	}

	return response.OK(c, "Tenant retrieved successfully", tenant)
}
