package middleware

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/utils/logger"
)

func LogMiddleware(l *logger.Logger, env string) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()

		err := c.Next()

		elapsed := time.Since(start)
		status := c.Response().StatusCode()
		method := c.Method()
		url := string(c.Request().URI().PathOriginal())
		remoteIP := c.IP()
		query := string(c.Request().URI().QueryString())

		userID, _ := c.Locals("user_id").(int)
		tenantID, _ := c.Locals("tenant_id").(int)

		var displayMsg string
		respBody := c.Response().Body()
		if len(respBody) > 0 && strings.Contains(string(c.Response().Header.ContentType()), "json") {
			var resp struct {
				Meta struct {
					Message string `json:"message"`
				} `json:"meta"`
			}
			if json.Unmarshal(respBody, &resp) == nil {
				displayMsg = resp.Meta.Message
			}
		}

		var reqBody string
		if env == "development" || status >= 400 {
			rawBody := c.Body()
			if len(rawBody) > 0 {
				ct := string(c.Request().Header.ContentType())
				if strings.Contains(ct, "json") {
					reqBody = truncate(prettifyJSON(string(rawBody)), 2000)
				} else if strings.Contains(ct, "text") || strings.Contains(ct, "form-urlencoded") {
					reqBody = truncate(string(rawBody), 2000)
				} else {
					reqBody = fmt.Sprintf("[%s %d bytes]", ct, len(rawBody))
				}
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

		authHeaders := map[string]string{}
		if auth := c.Get("Authorization"); auth != "" {
			authHeaders["authorization"] = truncate(auth, 120)
		}
		if key := c.Get("apikey"); key != "" {
			authHeaders["apikey"] = truncate(key, 40)
		}
		if key := c.Get("x-api-key"); key != "" {
			authHeaders["x-api-key"] = truncate(key, 40)
		}

		if l.Format() == "json" {
			log := map[string]interface{}{
				"timestamp":    ts,
				"level":        strings.ToLower(level),
				"context":      "HTTP",
				"method":       method,
				"url":          url,
				"status":       status,
				"responseTime": fmt.Sprintf("%dms", elapsed.Milliseconds()),
				"remoteIP":     remoteIP,
			}
			if query != "" {
				log["query"] = query
			}
			if userID > 0 {
				log["user_id"] = userID
			}
			if tenantID > 0 {
				log["tenant_id"] = tenantID
			}
			if displayMsg != "" {
				log["message"] = displayMsg
			}
			if len(authHeaders) > 0 {
				log["auth"] = authHeaders
			}
			if reqBody != "" {
				log["requestBody"] = reqBody
			}
			if env == "development" || status >= 500 {
				respBodyStr := string(respBody)
				if len(respBodyStr) > 0 {
					log["responseBody"] = truncate(respBodyStr, 2000)
				}
			}
			b, _ := json.MarshalIndent(log, "", "  ")
			fmt.Fprintln(os.Stdout, string(b))
		} else {
			fmt.Fprintf(os.Stdout, "\n")
			fmt.Fprintf(os.Stdout, "  \033[1m%s\033[0m  \033[97m%s\033[0m  %s\n", ts, padLevel(level), "\033[90mHTTP\033[0m")
			fmt.Fprintf(os.Stdout, "  \033[90mStatus\033[0m       %s\n", colorStatus)
			fmt.Fprintf(os.Stdout, "  \033[90mMethod\033[0m       %s\n", method)
			fmt.Fprintf(os.Stdout, "  \033[90mURL\033[0m          %s\n", url)
			if query != "" {
				fmt.Fprintf(os.Stdout, "  \033[90mQuery\033[0m         %s\n", query)
			}
			fmt.Fprintf(os.Stdout, "  \033[90mDuration\033[0m     %dms\n", elapsed.Milliseconds())
			fmt.Fprintf(os.Stdout, "  \033[90mRemote IP\033[0m    %s\n", remoteIP)
			if userID > 0 {
				fmt.Fprintf(os.Stdout, "  \033[90mUser ID\033[0m      %d\n", userID)
			}
			if tenantID > 0 {
				fmt.Fprintf(os.Stdout, "  \033[90mTenant ID\033[0m    %d\n", tenantID)
			}
			if displayMsg != "" {
				fmt.Fprintf(os.Stdout, "  \033[90mMessage\033[0m      %q\n", displayMsg)
			}
			for k, v := range authHeaders {
				fmt.Fprintf(os.Stdout, "  \033[90m%s\033[0m     %s\n", strings.ToUpper(k), v)
			}
			if reqBody != "" {
				fmt.Fprintf(os.Stdout, "  \033[90mRequest Body\033[0m %s\n", reqBody)
			}
			if env == "development" || status >= 500 {
				respBodyStr := string(respBody)
				if len(respBodyStr) > 0 {
					ct := string(c.Response().Header.ContentType())
					if strings.Contains(ct, "json") {
						respBodyStr = prettifyJSON(respBodyStr)
					}
					fmt.Fprintf(os.Stdout, "  \033[90mResponse Body\033[0m %s\n", truncate(respBodyStr, 2000))
				}
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

func prettifyJSON(s string) string {
	var v interface{}
	if json.Unmarshal([]byte(s), &v) == nil {
		truncateStrings(v)
		b, _ := json.MarshalIndent(v, "  ", "  ")
		if len(b) > 0 {
			return string(b)
		}
	}
	return s
}

func truncateStrings(v interface{}) {
	switch ref := v.(type) {
	case map[string]interface{}:
		for k, val := range ref {
			switch s := val.(type) {
			case string:
				if len(s) > 100 {
					ref[k] = fmt.Sprintf("[base64 %d chars]", len(s))
				}
			default:
				truncateStrings(val)
			}
		}
	case []interface{}:
		for _, val := range ref {
			truncateStrings(val)
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
