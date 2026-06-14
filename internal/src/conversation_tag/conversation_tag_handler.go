package conversation_tag

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
)

type ConversationTagHandler struct {
	service ConversationTagService
}

func NewConversationTagHandler(c *di.Container) *ConversationTagHandler {
	repo := NewConversationTagRepository()
	service := NewConversationTagService(repo, c.DB)
	return &ConversationTagHandler{service: service}
}

func (h *ConversationTagHandler) List(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	tags, err := h.service.ListByConversation(c.Context(), t.ID, conversationID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", tags)
}

func (h *ConversationTagHandler) Attach(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	var req AttachTagRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	if err := h.service.Attach(c.Context(), t.ID, conversationID, req.TagID); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Tag attached", nil)
}

func (h *ConversationTagHandler) Detach(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	conversationID, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	tagID, err := strconv.Atoi(c.Params("tagId"))
	if err != nil {
		return response.BadRequest(c, "Invalid tag ID", nil)
	}

	if err := h.service.Detach(c.Context(), t.ID, conversationID, tagID); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Tag detached", nil)
}
