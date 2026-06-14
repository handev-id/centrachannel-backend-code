package message

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
	"centrachannel/internal/ws"
)

type MessageHandler struct {
	service MessageService
}

func NewMessageHandler(c *di.Container, notifier ...ws.Notifier) *MessageHandler {
	repo := NewMessageRepository()
	convRepo := conversation.NewConversationRepository()
	profileRepo := profile.NewProfileRepository()
	channelRepo := channel.NewChannelRepository()
	tenantRepo := tenant.NewTenantRepository()
	service := NewMessageService(repo, convRepo, profileRepo, channelRepo, tenantRepo, c.DB, c.Config, c.Logger, notifier...)
	return &MessageHandler{service: service}
}

func NewMessageHandlerWithService(service MessageService) *MessageHandler {
	return &MessageHandler{service: service}
}

func (h *MessageHandler) List(c fiber.Ctx) error {
	conversationID, err := strconv.Atoi(c.Params("conversationId"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "50"))

	q := ListMessageQuery{Page: page, Limit: limit}
	result, err := h.service.List(c.Context(), conversationID, q)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	return response.OK(c, "success", result)
}

func (h *MessageHandler) Send(c fiber.Ctx, t *tenant.Tenant) error {

	conversationID, err := strconv.Atoi(c.Params("conversationId"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	var req SendMessageRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	msg, err := h.service.Send(c.Context(), req, t.ID, conversationID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Message sent", msg)
}

func (h *MessageHandler) UpdateStatus(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateMessageStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	if err := h.service.UpdateStatus(c.Context(), id, req.Status); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Message status updated", nil)
}
