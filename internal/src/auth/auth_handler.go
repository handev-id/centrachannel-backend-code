package auth

import (
	"github.com/gofiber/fiber/v3"

	"centrachannel/config"
	"centrachannel/internal/di"
	"centrachannel/internal/middleware"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

type AuthHandler struct {
	service AuthService
	cfg     *config.Config
}

func NewAuthHandler(container *di.Container) *AuthHandler {
	repo := NewAuthRepository()
	service := NewAuthService(repo, container.DB, container.Config, container.Logger, container.Redis)
	return &AuthHandler{service: service, cfg: container.Config}
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

	if h.cfg != nil {
		secure := h.cfg.Env == "production"
		c.Cookie(&fiber.Cookie{
			Name:     middleware.TokenCookieName,
			Value:    token,
			Path:     "/",
			MaxAge:   int(h.cfg.JWTExpiry.Seconds()),
			HTTPOnly: true,
			Secure:   secure,
			SameSite: fiber.CookieSameSiteStrictMode,
		})
	}

	return response.OK(c, "Login successful", nil)
}

func (h *AuthHandler) CheckToken(c fiber.Ctx, t *tenant.Tenant) error {
	token := middleware.ExtractToken(c)
	if token == "" {
		return response.Unauthorized(c, "Missing token")
	}
	user, err := h.service.CheckToken(c.Context(), token, t)
	if err != nil {
		return response.Unauthorized(c, err.Error())
	}
	return response.OK(c, "Token valid", user)
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	tokenStr := middleware.ExtractToken(c)
	if tokenStr == "" {
		return response.Unauthorized(c, "Missing token")
	}

	if err := h.service.Logout(c.Context(), tokenStr); err != nil {
		return response.Unauthorized(c, err.Error())
	}

	c.Cookie(&fiber.Cookie{
		Name:     middleware.TokenCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteStrictMode,
	})

	return response.OK(c, "Logged out successfully", nil)
}
