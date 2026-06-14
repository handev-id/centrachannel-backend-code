package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"

	"centrachannel/config"
	"centrachannel/internal/utils/response"
)

func AuthMiddleware(cfg *config.Config) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return response.Unauthorized(c, "Missing token")
		}

		tokenStr := authHeader
		if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = authHeader[7:]
		}

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			return response.Unauthorized(c, "Invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return response.Unauthorized(c, "Invalid token claims")
		}

		subFloat, _ := claims["sub"].(float64)
		tenantFloat, _ := claims["tenant"].(float64)

		c.Locals("user", claims)
		c.Locals("user_id", int(subFloat))
		c.Locals("tenant_id", int(tenantFloat))
		return c.Next()
	}
}
