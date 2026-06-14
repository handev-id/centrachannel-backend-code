package note

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

type NoteHandler struct {
	service NoteService
}

func NewNoteHandler(c *di.Container) *NoteHandler {
	repo := NewNoteRepository()
	service := NewNoteService(repo, c.DB, c.Config, c.Logger)
	return &NoteHandler{service: service}
}

func NewNoteHandlerWithService(service NoteService) *NoteHandler {
	return &NoteHandler{service: service}
}

func (h *NoteHandler) List(c fiber.Ctx, t *tenant.Tenant) error {

	conversationID, err := strconv.Atoi(c.Params("conversationId"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	notes, err := h.service.ListByConversation(c.Context(), t.ID, conversationID)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", notes)
}

func (h *NoteHandler) Store(c fiber.Ctx, t *tenant.Tenant) error {
	userID, err := middleware.GetUserID(c)
	if err != nil { return response.Unauthorized(c, err.Error()) }

	conversationID, err := strconv.Atoi(c.Params("conversationId"))
	if err != nil {
		return response.BadRequest(c, "Invalid conversation ID", nil)
	}

	var req CreateNoteRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	note, err := h.service.Create(c.Context(), req, t.ID, conversationID, userID)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "Note created", note)
}

func (h *NoteHandler) Update(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateNoteRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	note, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Note updated", note)
}

func (h *NoteHandler) Delete(c fiber.Ctx, t *tenant.Tenant) error {

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Delete(c.Context(), t.ID, id); err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "Note deleted", nil)
}
