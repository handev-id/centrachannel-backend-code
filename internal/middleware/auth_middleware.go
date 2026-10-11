package middleware

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/response"
)

func AuthMiddleware(cfg *config.Config, rdb *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		tokenStr := ExtractToken(c)

		if tokenStr == "" {
			return response.Unauthorized(c, "Missing token")
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

		sub, subOK := claims["sub"].(string)
		tenantClaim, tenantOK := claims["tenant"].(string)
		userID, userErr := uuid.Parse(sub)
		tenantID, tenantErr := uuid.Parse(tenantClaim)
		currentTenant, currentTenantOK := c.Locals("tenant").(*tenant.Tenant)
		if !subOK || !tenantOK || userErr != nil || tenantErr != nil || userID == uuid.Nil || tenantID == uuid.Nil || !currentTenantOK || currentTenant == nil || tenantID != currentTenant.ID {
			return response.Unauthorized(c, "Invalid token tenant")
		}

		rawRoles, _ := claims["roles"].([]interface{})
		roles := make([]string, len(rawRoles))
		for i, r := range rawRoles {
			roles[i], _ = r.(string)
		}

		c.Locals("user", claims)
		c.Locals("user_id", userID)
		c.Locals("tenant_id", tenantID)
		c.Locals("roles", roles)
		return c.Next()
	}
}
