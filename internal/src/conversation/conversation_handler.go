package conversation

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
	"centrachannel/internal/ws"
)

type ConversationHandler struct {
	service ConversationService
}

func NewConversationHandler(c *di.Container, notifier ...ws.Notifier) *ConversationHandler {
	repo := NewConversationRepository()
	service := NewConversationService(repo, c.DB, c.Config, c.Logger, notifier...)
	return &ConversationHandler{service: service}
}

func NewConversationHandlerWithService(service ConversationService) *ConversationHandler {
	return &ConversationHandler{service: service}
}

func (h *ConversationHandler) List(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	agentID, _ := strconv.Atoi(c.Query("agent_id"))

	q := ListConversationQuery{
		Page: page, Limit: limit, Status: c.Query("status"),
		ChannelID: channelID, AgentID: agentID, Search: c.Query("search"),
		SortBy: c.Query("sort_by"),
	}

	result, err := h.service.List(c.Context(), q, t)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", result)
}

func (h *ConversationHandler) Show(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	conv, err := h.service.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, "success", conv)
}

func (h *ConversationHandler) Store(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	var req CreateConversationRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	conv, err := h.service.Create(c.Context(), req, t)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Conversation created", conv)
}

func (h *ConversationHandler) Assign(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	userID, err := middleware.GetUserID(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Assign(c.Context(), t.ID, id, userID); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Conversation assigned", nil)
}

func (h *ConversationHandler) Unassign(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Unassign(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Conversation unassigned", nil)
}

func (h *ConversationHandler) Resolve(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Resolve(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Conversation resolved", nil)
}

func (h *ConversationHandler) Reopen(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Reopen(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Conversation reopened", nil)
}

func (h *ConversationHandler) Read(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.MarkRead(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Conversation marked as read", nil)
}
