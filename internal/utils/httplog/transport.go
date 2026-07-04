package httplog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"centrachannel/internal/utils/logger"
)

type LoggingRoundTripper struct {
	next   http.RoundTripper
	logger *logger.Logger
	name   string
}

func NewLoggingRoundTripper(next http.RoundTripper, l *logger.Logger, name string) http.RoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &LoggingRoundTripper{next: next, logger: l, name: name}
}

func NewLoggingClient(l *logger.Logger, name string) *http.Client {
	return &http.Client{
		Transport: NewLoggingRoundTripper(http.DefaultTransport, l, name),
	}
}

func (t *LoggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	reqBody := ""
	if req.Body != nil && req.Body != http.NoBody {
		bodyBytes, _ := io.ReadAll(req.Body)
		req.Body.Close()
		if len(bodyBytes) > 0 {
			ct := req.Header.Get("Content-Type")
			if strings.Contains(ct, "json") || strings.Contains(ct, "text") || strings.Contains(ct, "form") {
				reqBody = truncate(string(bodyBytes), 2000)
			} else {
				reqBody = fmt.Sprintf("[%s %d bytes]", ct, len(bodyBytes))
			}
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	resp, err := t.next.RoundTrip(req)
	elapsed := time.Since(start)

	respBody := ""
	respStatus := 0
	respContentType := ""
	if err == nil && resp != nil {
		respStatus = resp.StatusCode
		respContentType = resp.Header.Get("Content-Type")
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if len(bodyBytes) > 0 && (strings.Contains(respContentType, "json") || strings.Contains(respContentType, "text")) {
			respBody = truncate(string(bodyBytes), 2000)
		}
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	authHeaders := map[string]string{}
	if auth := req.Header.Get("Authorization"); auth != "" {
		authHeaders["authorization"] = truncate(auth, 120)
	}
	if key := req.Header.Get("apikey"); key != "" {
		authHeaders["apikey"] = truncate(key, 40)
	}
	if key := req.Header.Get("x-api-key"); key != "" {
		authHeaders["x-api-key"] = truncate(key, 40)
	}

	level := "INFO"
	if err != nil {
		level = "ERROR"
	} else if respStatus >= 500 {
		level = "ERROR"
	} else if respStatus >= 400 {
		level = "WARN"
	}

	logFields := map[string]interface{}{
		"context":      "HTTP",
		"direction":    "OUTBOUND",
		"service":      t.name,
		"method":       req.Method,
		"url":          req.URL.String(),
		"endpoint":     req.URL.Path,
		"status":       respStatus,
		"responseTime": fmt.Sprintf("%dms", elapsed.Milliseconds()),
	}
	if err != nil {
		logFields["error"] = err.Error()
	}
	if len(authHeaders) > 0 {
		logFields["auth"] = authHeaders
	}
	if reqBody != "" {
		logFields["requestBody"] = reqBody
	}
	if respBody != "" {
		logFields["responseBody"] = respBody
	}

	logJSON(level, logFields, t.logger)

	return resp, err
}

func logJSON(level string, data map[string]interface{}, l *logger.Logger) {
	b, _ := json.MarshalIndent(data, "", "  ")
	msg := string(b)
	switch level {
	case "ERROR":
		l.Error(msg)
	case "WARN":
		l.Warn(msg)
	default:
		l.Info(msg)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
