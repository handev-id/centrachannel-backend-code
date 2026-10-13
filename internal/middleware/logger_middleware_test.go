package middleware

import (
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/utils/logger"
)

func TestLogMiddlewareEndpointWhitelist(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		path    string
		status  int
		failure bool
		hidden  bool
	}{
		{name: "healthy", method: "GET", path: "/health", status: 200, hidden: true},
		{name: "healthy with query", method: "GET", path: "/health?probe=docker", status: 200, hidden: true},
		{name: "healthy no content", method: "GET", path: "/health", status: 204, hidden: true},
		{name: "redirect", method: "GET", path: "/health", status: 302},
		{name: "client failure", method: "GET", path: "/health", status: 400},
		{name: "server failure", method: "GET", path: "/health", status: 503},
		{name: "returned error", method: "GET", path: "/health", status: 503, failure: true},
		{name: "other method", method: "POST", path: "/health", status: 200},
		{name: "other endpoint", method: "GET", path: "/ping", status: 200},
		{name: "nested endpoint", method: "GET", path: "/health/details", status: 200},
		{name: "prefixed endpoint", method: "GET", path: "/api/health", status: 200},
	}
	for _, format := range []string{"json", "text"} {
		for _, tc := range cases {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				app := fiber.New()
				app.Use(LogMiddleware(logger.NewLogger("info", format), "production"))
				app.Use(func(c fiber.Ctx) error {
					if tc.failure {
						return fiber.NewError(tc.status, "probe failed")
					}
					return c.SendStatus(tc.status)
				})

				r, w, err := os.Pipe()
				if err != nil {
					t.Fatal(err)
				}
				stdout := os.Stdout
				os.Stdout = w
				defer func() {
					os.Stdout = stdout
					w.Close()
					r.Close()
				}()

				resp, err := app.Test(httptest.NewRequest(tc.method, tc.path, nil))
				os.Stdout = stdout
				w.Close()
				if err != nil {
					t.Fatal(err)
				}
				defer resp.Body.Close()
				if resp.StatusCode != tc.status {
					t.Fatalf("status = %d, want %d", resp.StatusCode, tc.status)
				}
				output, err := io.ReadAll(r)
				if err != nil {
					t.Fatal(err)
				}
				if tc.hidden && len(output) != 0 {
					t.Fatalf("expected no log, got %s", output)
				}
				if !tc.hidden && !strings.Contains(string(output), "HTTP") {
					t.Fatalf("expected HTTP log, got %q", output)
				}
			})
		}
	}
}
