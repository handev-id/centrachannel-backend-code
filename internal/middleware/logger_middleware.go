package middleware

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/utils/logger"
	"centrachannel/internal/utils/response"
)

func LogMiddleware(l *logger.Logger, env string) fiber.Handler {
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

		level := "INFO"
		if status >= 500 {
			level = "ERROR"
		} else if status >= 400 {
			level = "WARN"
		}

		ts := time.Now().Format("2006-01-02 15:04:05")
		colorStatus := colorStatusFn(status)

		if l.Format() == "json" {
			log := map[string]interface{}{
				"timestamp":     ts,
				"level":         strings.ToLower(level),
				"context":       "HTTP",
				"method":        method,
				"url":           url,
				"status":        status,
				"responseTime":  fmt.Sprintf("%dms", elapsed.Milliseconds()),
			}
			if userID > 0 {
				log["user_id"] = userID
			}
			if displayMsg != "" {
				log["message"] = displayMsg
			}
			if env == "development" {
				if auth := c.Get("Authorization"); auth != "" {
					log["authorization"] = truncate(auth, 80)
				}
			}
			if status >= 500 && len(systemMsg) > 0 {
				log["body"] = string(systemMsg)
			}
			b, _ := json.MarshalIndent(log, "", "  ")
			fmt.Fprintln(os.Stdout, string(b))
		} else {
			fmt.Fprintf(os.Stdout, "\n")
			fmt.Fprintf(os.Stdout, "  \033[1m%s\033[0m  \033[97m%s\033[0m  %s\n", ts, padLevel(level), "\033[90mHTTP\033[0m")
			fmt.Fprintf(os.Stdout, "  \033[90mStatus\033[0m       %s\n", colorStatus)
			fmt.Fprintf(os.Stdout, "  \033[90mMethod\033[0m       %s\n", method)
			fmt.Fprintf(os.Stdout, "  \033[90mURL\033[0m          %s\n", url)
			fmt.Fprintf(os.Stdout, "  \033[90mDuration\033[0m     %s %s\n", elapsed.Round(time.Millisecond), "\033[90mms\033[0m")
			if userID > 0 {
				fmt.Fprintf(os.Stdout, "  \033[90mUser ID\033[0m      %d\n", userID)
			}
			if displayMsg != "" {
				fmt.Fprintf(os.Stdout, "  \033[90mMessage\033[0m      %q\n", displayMsg)
			}
			if env == "development" {
				if auth := c.Get("Authorization"); auth != "" {
					fmt.Fprintf(os.Stdout, "  \033[90mAuthorization\033[0m %s\n", truncate(auth, 80))
				}
			}
			if status >= 500 && len(systemMsg) > 0 {
				fmt.Fprintf(os.Stdout, "  \033[90mBody\033[0m          %s\n", string(systemMsg))
			}
			fmt.Fprintf(os.Stdout, "\n")
		}

		return err
	}
}

func padLevel(level string) string {
	return fmt.Sprintf("%-5s", level)
}

func colorStatusFn(status int) string {
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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
