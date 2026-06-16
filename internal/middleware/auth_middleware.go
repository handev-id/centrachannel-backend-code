package middleware

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"centrachannel/config"
	"centrachannel/internal/utils/response"
)

func AuthMiddleware(cfg *config.Config, rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		tokenStr := authHeader

		if authHeader == "" {
			tokenStr = c.Query("token")
		}
		
		if tokenStr == "" {
			return response.Unauthorized(c, "Missing token")
		}

		if len(tokenStr) > 7 && strings.HasPrefix(tokenStr, "Bearer ") {
			tokenStr = tokenStr[7:]
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

		if jti, _ := claims["jti"].(string); jti != "" && rdb != nil {
			blacklisted, err := rdb.Exists(c.Context(), "token_blacklist:"+jti).Result()
			if err == nil && blacklisted > 0 {
				return response.Unauthorized(c, "Token has been revoked")
			}
		}

		subFloat, _ := claims["sub"].(float64)
		tenantFloat, _ := claims["tenant"].(float64)

		rawRoles, _ := claims["roles"].([]interface{})
		roles := make([]string, len(rawRoles))
		for i, r := range rawRoles {
			roles[i], _ = r.(string)
		}

		c.Locals("user", claims)
		c.Locals("user_id", int(subFloat))
		c.Locals("tenant_id", int(tenantFloat))
		c.Locals("roles", roles)
		return c.Next()
	}
}
