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

func NewMessageHandler(c *di.Container) *MessageHandler {
	repo := NewMessageRepository()
	convRepo := conversation.NewConversationRepository()
	profileRepo := profile.NewProfileRepository()
	channelRepo := channel.NewChannelRepository()
	tenantRepo := tenant.NewTenantRepository()
	service := NewMessageService(repo, convRepo, profileRepo, channelRepo, tenantRepo, c.DB, c.Config, c.Logger, c.Redis, ws.NewHubNotifier(c.Hub))
	return &MessageHandler{service: service}
}

func NewMessageHandlerWithService(service MessageService) *MessageHandler {
	return &MessageHandler{service: service}
}

func (h *MessageHandler) List(c fiber.Ctx, t *tenant.Tenant) error {
	conversationID, err := strconv.Atoi(c.Query("conversation_id"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	lastID, _ := strconv.Atoi(c.Query("last_id"))

	q := ListMessageQuery{Limit: limit, LastID: lastID}

	msgs, lastID, hasMore, err := h.service.ListCursor(c.Context(), t.ID, conversationID, q)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.CursorPaginated(c, "success", msgs, lastID, hasMore)
}

func (h *MessageHandler) Send(c fiber.Ctx, t *tenant.Tenant) error {

	var req SendMessageRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	msg, err := h.service.Send(c.Context(), req, t.ID, req.ConversationID)
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
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	if err := h.service.UpdateStatus(c.Context(), id, req.Status); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Message status updated", nil)
}
