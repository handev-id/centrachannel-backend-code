package tenant

import (
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

func (h *TenantHandler) Get(c fiber.Ctx) error {
	t, ok := c.Locals("tenant").(*Tenant)
	if !ok || t == nil {
		return response.InternalServerError(c, "tenant context not found")
	}
	return response.OK(c, "Tenant retrieved successfully", t)
}

func (h *TenantHandler) Update(c fiber.Ctx) error {
	t, ok := c.Locals("tenant").(*Tenant)
	if !ok || t == nil {
		return response.InternalServerError(c, "tenant context not found")
	}

	var req UpdateTenantRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	if req.Name != nil {
		t.Name = *req.Name
	}
	if req.Logo != nil {
		t.Logo = req.Logo
	}
	if req.Address != nil {
		t.Address = req.Address
	}
	if req.Phone != nil {
		t.Phone = req.Phone
	}
	if req.Email != nil {
		t.Email = req.Email
	}
	if req.IsActive != nil {
		t.IsActive = *req.IsActive
	}
	if req.Settings != nil {
		t.Settings = req.Settings
	}

	if err := h.service.Update(c.Context(), t); err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "Tenant updated successfully", t)
}
