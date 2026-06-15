package middleware

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/utils/logger"
	"centrachannel/internal/utils/response"
)

func NewLogMiddleware(l *logger.Logger, env string) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		elapsed := time.Since(start)
		status := c.Response().StatusCode()
		method := c.Method()
		url := string(c.Request().URI().PathOriginal())

		userID, _ := c.Locals("user_id").(int)

		var displayMsg string
		var systemMsg json.RawMessage

		respBody := c.Response().Body()
		if len(respBody) > 0 && strings.Contains(string(c.Response().Header.ContentType()), "json") {
			var resp response.Response
			if json.Unmarshal(respBody, &resp) == nil {
				displayMsg = resp.Meta.Message
			}
			if status >= 500 {
				systemMsg = json.RawMessage(respBody)
			}
		}

		msg := fmt.Sprintf("%s %s %s %dms",
			padMethod(method),
			url,
			colorStatus(status),
			elapsed.Milliseconds(),
		)

		if userID > 0 {
			msg += fmt.Sprintf(" user:%d", userID)
		}
		if displayMsg != "" {
			msg += fmt.Sprintf(" %q", displayMsg)
		}
		if env == "development" {
			auth := c.Get("Authorization")
			if auth != "" {
				if len(auth) > 80 {
					auth = auth[:80] + "..."
				}
				msg += fmt.Sprintf(" Authorization:%s", auth)
			}
		}
		if status >= 500 && len(systemMsg) > 0 {
			msg += fmt.Sprintf(" body:%s", string(systemMsg))
		}

		logFn := l.Info
		if status >= 500 {
			logFn = l.Error
		} else if status >= 400 {
			logFn = l.Warn
		}
		logFn("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), msg)

		return err
	}
}

func padMethod(method string) string {
	return fmt.Sprintf("%-6s", method)
}

func colorStatus(status int) string {
	code := fmt.Sprintf("%d", status)
	switch {
	case status >= 500:
		return "\033[31m" + code + "\033[0m"
	case status >= 400:
		return "\033[33m" + code + "\033[0m"
	case status >= 300:
		return "\033[36m" + code + "\033[0m"
	default:
		return "\033[32m" + code + "\033[0m"
	}
}
