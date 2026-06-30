package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
)

const TokenCookieName = "CrmToken"

func ExtractToken(c fiber.Ctx) string {
	tokenStr := c.Cookies(TokenCookieName)
	if tokenStr == "" {
		tokenStr = c.Get("Authorization")
	}
	if tokenStr == "" {
		tokenStr = c.Query("token")
	}
	if len(tokenStr) > 7 && strings.HasPrefix(tokenStr, "Bearer ") {
		tokenStr = tokenStr[7:]
	}
	return tokenStr
}
