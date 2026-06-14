package campaign

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/response"
)

type CampaignHandler struct {
	service CampaignService
}

func NewCampaignHandler(c *di.Container) *CampaignHandler {
	repo := NewCampaignRepository()
	deviceRepo := whatsapp_device.NewWhatsAppDeviceRepository()
	channelRepo := channel.NewChannelRepository()
	service := NewCampaignService(repo, deviceRepo, channelRepo, c.DB, c.Config, c.Logger)
	return &CampaignHandler{service: service}
}

func NewCampaignHandlerWithService(service CampaignService) *CampaignHandler {
	return &CampaignHandler{service: service}
}

func (h *CampaignHandler) List(c fiber.Ctx, t *tenant.Tenant) error {

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	result, err := h.service.List(c.Context(), ListCampaignQuery{Page: page, Limit: limit, Search: c.Query("search")}, t)
	if err != nil { return response.InternalServerError(c, err.Error()) }
	return response.OK(c, "success", result)
}

func (h *CampaignHandler) Show(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	campaign, err := h.service.GetByID(c.Context(), t.ID, id)
	if err != nil { return response.NotFound(c, err.Error()) }
	return response.OK(c, "success", campaign)
}

func (h *CampaignHandler) Store(c fiber.Ctx, t *tenant.Tenant) error {
	userID, err := middleware.GetUserID(c)
	if err != nil { return response.Unauthorized(c, err.Error()) }

	var req CreateCampaignRequest
	if err := c.Bind().Body(&req); err != nil { return response.BadRequest(c, "Invalid payload", nil) }
	if err := response.Validate(c, &req); err != nil { return err }

	campaign, err := h.service.Create(c.Context(), req, t, userID)
	if err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.Created(c, "Campaign created", campaign)
}

func (h *CampaignHandler) Update(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	var req UpdateCampaignRequest
	if err := c.Bind().Body(&req); err != nil { return response.BadRequest(c, "Invalid payload", nil) }
	if err := response.Validate(c, &req); err != nil { return err }

	campaign, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.OK(c, "Campaign updated", campaign)
}

func (h *CampaignHandler) Delete(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	if err := h.service.Delete(c.Context(), t.ID, id); err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.OK(c, "Campaign deleted", nil)
}

func (h *CampaignHandler) Send(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	if err := h.service.Send(c.Context(), t.ID, id); err != nil { return response.InternalServerError(c, err.Error()) }
	return response.OK(c, "Campaign send started", nil)
}

func (h *CampaignHandler) ListTemplates(c fiber.Ctx, t *tenant.Tenant) error {

	templates, err := h.service.ListTemplates(c.Context(), t.ID)
	if err != nil { return response.InternalServerError(c, err.Error()) }
	return response.OK(c, "success", templates)
}

func (h *CampaignHandler) GetTemplate(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	template, err := h.service.GetTemplateByID(c.Context(), t.ID, id)
	if err != nil { return response.NotFound(c, err.Error()) }
	return response.OK(c, "success", template)
}

func (h *CampaignHandler) CreateTemplate(c fiber.Ctx, t *tenant.Tenant) error {

	var req CreateTemplateRequest
	if err := c.Bind().Body(&req); err != nil { return response.BadRequest(c, "Invalid payload", nil) }
	if err := response.Validate(c, &req); err != nil { return err }

	template, err := h.service.CreateTemplate(c.Context(), req, t)
	if err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.Created(c, "Template created", template)
}

func (h *CampaignHandler) UpdateTemplate(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	var req UpdateTemplateRequest
	if err := c.Bind().Body(&req); err != nil { return response.BadRequest(c, "Invalid payload", nil) }
	if err := response.Validate(c, &req); err != nil { return err }

	template, err := h.service.UpdateTemplate(c.Context(), t.ID, id, req)
	if err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.OK(c, "Template updated", template)
}

func (h *CampaignHandler) DeleteTemplate(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	if err := h.service.DeleteTemplate(c.Context(), t.ID, id); err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.OK(c, "Template deleted", nil)
}

func (h *CampaignHandler) ListRecipientLists(c fiber.Ctx, t *tenant.Tenant) error {

	lists, err := h.service.ListRecipientLists(c.Context(), t.ID)
	if err != nil { return response.InternalServerError(c, err.Error()) }
	return response.OK(c, "success", lists)
}

func (h *CampaignHandler) GetRecipientList(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	list, err := h.service.GetRecipientListByID(c.Context(), t.ID, id)
	if err != nil { return response.NotFound(c, err.Error()) }
	return response.OK(c, "success", list)
}

func (h *CampaignHandler) CreateRecipientList(c fiber.Ctx, t *tenant.Tenant) error {

	var req CreateRecipientListRequest
	if err := c.Bind().Body(&req); err != nil { return response.BadRequest(c, "Invalid payload", nil) }
	if err := response.Validate(c, &req); err != nil { return err }

	list, err := h.service.CreateRecipientList(c.Context(), req, t)
	if err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.Created(c, "Recipient list created", list)
}

func (h *CampaignHandler) UpdateRecipientList(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	var req UpdateRecipientListRequest
	if err := c.Bind().Body(&req); err != nil { return response.BadRequest(c, "Invalid payload", nil) }
	if err := response.Validate(c, &req); err != nil { return err }

	list, err := h.service.UpdateRecipientList(c.Context(), t.ID, id, req)
	if err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.OK(c, "Recipient list updated", list)
}

func (h *CampaignHandler) DeleteRecipientList(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid ID", nil) }

	if err := h.service.DeleteRecipientList(c.Context(), t.ID, id); err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.OK(c, "Recipient list deleted", nil)
}

func (h *CampaignHandler) ListRecipientContacts(c fiber.Ctx) error {
	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid list ID", nil) }

	contacts, err := h.service.ListRecipientContacts(c.Context(), listID)
	if err != nil { return response.InternalServerError(c, err.Error()) }
	return response.OK(c, "success", contacts)
}

func (h *CampaignHandler) AddRecipientContact(c fiber.Ctx) error {
	listID, err := strconv.Atoi(c.Params("id"))
	if err != nil { return response.BadRequest(c, "Invalid list ID", nil) }

	var req AddContactToListRequest
	if err := c.Bind().Body(&req); err != nil { return response.BadRequest(c, "Invalid payload", nil) }
	if err := response.Validate(c, &req); err != nil { return err }

	contact, err := h.service.AddRecipientContact(c.Context(), listID, req)
	if err != nil { return response.BadRequest(c, err.Error(), nil) }
	return response.Created(c, "Contact added to list", contact)
}

func (h *CampaignHandler) RemoveRecipientContact(c fiber.Ctx) error {
	contactID, err := strconv.Atoi(c.Params("contactId"))
	if err != nil { return response.BadRequest(c, "Invalid contact ID", nil) }

	if err := h.service.RemoveRecipientContact(c.Context(), contactID); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Contact removed from list", nil)
}
