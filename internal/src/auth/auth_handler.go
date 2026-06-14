package auth

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/di"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

type AuthHandler struct {
	service AuthService
}

func NewAuthHandler(container *di.Container) *AuthHandler {
	repo := NewAuthRepository()
	service := NewAuthService(repo, container.DB, container.Config, container.Logger, container.Redis)
	return &AuthHandler{service: service}
}

func NewAuthHandlerWithService(service AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c fiber.Ctx, t *tenant.Tenant) error {
	var req RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}

	user, err := h.service.Register(c.Context(), req, t)
	if err != nil {
		return response.BadRequest(c, err.Error(), nil)
	}
	return response.Created(c, "User registered", user)
}

func (h *AuthHandler) Login(c fiber.Ctx, t *tenant.Tenant) error {
	var req LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return response.BadRequest(c, "Invalid payload", nil)
	}
	if err := response.Validate(c, &req); err != nil {
		return err
	}
	token, err := h.service.Login(c.Context(), req, t)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}
	return response.OK(c, "Login successful", fiber.Map{"type": "bearer", "token": token})
}

func (h *AuthHandler) CheckToken(c fiber.Ctx, t *tenant.Tenant) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return response.Unauthorized(c, "Missing token")
	}
	token := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}
	user, err := h.service.CheckToken(c.Context(), token, t)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}
	return response.OK(c, "Token valid", user)
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return response.Unauthorized(c, "Missing token")
	}
	tokenStr := authHeader
	if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = authHeader[7:]
	}

	if err := h.service.Logout(c.Context(), tokenStr); err != nil {
		return response.Unauthorized(c, err.Error())
	}
	return response.OK(c, "Logged out successfully", nil)
}
