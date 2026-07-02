package whatsapp_device

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

type WhatsAppDeviceHandler struct {
	service WhatsAppDeviceService
}

func NewWhatsAppDeviceHandler(c *di.Container) *WhatsAppDeviceHandler {
	repo := NewWhatsAppDeviceRepository()
	client := NewEvolutionClient(c.Config.EvolutionAPIURL, c.Config.EvolutionAPIKey, c.Logger)
	service := NewWhatsAppDeviceService(repo, c.DB, c.Config, c.Logger, client)
	return &WhatsAppDeviceHandler{service: service}
}

func NewWhatsAppDeviceHandlerWithService(service WhatsAppDeviceService) *WhatsAppDeviceHandler {
	return &WhatsAppDeviceHandler{service: service}
}

func (h *WhatsAppDeviceHandler) Show(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	device, err := h.service.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, "success", device)
}

func (h *WhatsAppDeviceHandler) List(c fiber.Ctx, t *tenant.Tenant) error {

	devices, err := h.service.List(c.Context(), t.ID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", devices)
}

func (h *WhatsAppDeviceHandler) Store(c fiber.Ctx, t *tenant.Tenant) error {

	var req CreateDeviceRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	device, err := h.service.Create(c.Context(), req, t.ID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Device created", device)
}

func (h *WhatsAppDeviceHandler) Update(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateDeviceRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	device, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Device updated", device)
}

func (h *WhatsAppDeviceHandler) Delete(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Delete(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Device deleted", nil)
}

func (h *WhatsAppDeviceHandler) Connect(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	pairingCode, err := h.service.Connect(c.Context(), t.ID, id)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "success", fiber.Map{"pairing_code": pairingCode})
}

func (h *WhatsAppDeviceHandler) Disconnect(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	device, err := h.service.Disconnect(c.Context(), t.ID, id)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Device disconnected", device)
}

func (h *WhatsAppDeviceHandler) ConnectionState(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	connected, err := h.service.CheckConnection(c.Context(), t.ID, id)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "success", fiber.Map{"connected": connected})
}

func (h *WhatsAppDeviceHandler) Scan(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	qrCode, err := h.service.Scan(c.Context(), t.ID, id)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "success", fiber.Map{"qr_code": qrCode})
}
