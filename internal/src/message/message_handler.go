package message

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

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
	rawID := c.Params("conversationId")
	if rawID == "" {
		rawID = c.Query("conversation_id")
	}
	conversationID, err := uuid.Parse(rawID)
	if err != nil || conversationID == uuid.Nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	var lastID uuid.UUID
	if raw := c.Query("last_id"); raw != "" {
		var err error
		lastID, err = uuid.Parse(raw)
		if err != nil || lastID == uuid.Nil {
			return response.BadRequest(c, "Invalid last_id", nil)
		}
	}

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
	if rawID := c.Params("conversationId"); rawID != "" {
		id, err := uuid.Parse(rawID)
		if err != nil || id == uuid.Nil {
			return response.BadRequest(c, "Invalid conversation ID", nil)
		}
		if req.ConversationID != uuid.Nil && req.ConversationID != id {
			return response.BadRequest(c, "Conversation ID does not match route", nil)
		}
		req.ConversationID = id
	}
	if userID, ok := c.Locals("user_id").(uuid.UUID); ok && userID != uuid.Nil {
		req.SenderID = userID
		req.SenderType = "user"
	}
	if err := response.Validate(&req); err != nil {
		return response.ValidationFailed(c, err)
	}

	msg, err := h.service.Send(c.Context(), req, t.ID, req.ConversationID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Message queued", msg)
}

func (h *MessageHandler) UpdateStatus(c fiber.Ctx, t *tenant.Tenant) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateMessageStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(&req); err != nil {
		return response.ValidationFailed(c, err)
	}

	if err := h.service.UpdateStatus(c.Context(), t.ID, id, req.Status); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Message status updated", nil)
}

func (h *MessageHandler) Retry(c fiber.Ctx, t *tenant.Tenant) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil || id == uuid.Nil {
		return response.BadRequest(c, "Invalid message ID", nil)
	}
	userID, ok := c.Locals("user_id").(uuid.UUID)
	if !ok || userID == uuid.Nil {
		return response.Unauthorized(c, "Missing user")
	}
	msg, err := h.service.Retry(c.Context(), t.ID, id, userID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Message queued for retry", msg)
}
