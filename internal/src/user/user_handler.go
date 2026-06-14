package user

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/utils/response"
)

type UserHandler struct {
	service UserService
}

func NewUserHandler(container *di.Container) *UserHandler {
	repo := NewUserRepository()
	service := NewUserService(repo, container.DB, container.Config, container.Logger)
	return &UserHandler{service: service}
}

func (h *UserHandler) List(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))

	var roleID *int
	if rid := c.Query("role_id"); rid != "" {
		if v, err := strconv.Atoi(rid); err == nil {
			roleID = &v
		}
	}

	q := ListUserQuery{
		Page:   page,
		Limit:  limit,
		Search: c.Query("search"),
		RoleID: roleID,
		SortBy: c.Query("sort_by"),
	}

	result, err := h.service.List(c.Context(), q, t)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}
	return response.OK(c, "success", result)
}

func (h *UserHandler) Show(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	user, err := h.service.GetByID(c.Context(), t.ID, id)
	if err != nil {
		return response.NotFound(c, err.Error())
	}
	return response.OK(c, "success", user)
}

func (h *UserHandler) Store(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	var req CreateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	user, err := h.service.Create(c.Context(), req, t)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "User created", user)
}

func (h *UserHandler) Update(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	var req UpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}

	user, err := h.service.Update(c.Context(), t.ID, id, req)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "User updated", user)
}

func (h *UserHandler) Delete(c fiber.Ctx) error {
	t, err := middleware.GetTenant(c)
	if err != nil {
		return response.InternalServerError(c, err.Error())
	}

	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return response.BadRequest(c, "Invalid ID", nil)
	}

	if err := h.service.Delete(c.Context(), t.ID, id); err != nil {
		if err.Error() == "forbidden: you can not delete super admin" {
			return response.Forbidden(c, err.Error())
		}
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.OK(c, "User deleted successfully", nil)
}
