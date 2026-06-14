package integration

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"strings"

	"github.com/gofiber/fiber/v3"

	"centrachannel/internal/middleware"
	"centrachannel/internal/src/tenant"
)

func TestAuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("tenant", &tenant.Tenant{ID: 1, Name: "Test Tenant", Domain: "test.com"})
		c.Locals("user_id", 1)
		return c.Next()
	}
}

func NewTestApp() *fiber.App {
	return fiber.New(fiber.Config{})
}

func JSONBody(v interface{}) io.Reader {
	data, _ := json.Marshal(v)
	return strings.NewReader(string(data))
}

func ParseJSON(resp *httptest.ResponseRecorder, v interface{}) error {
	body, _ := io.ReadAll(resp.Result().Body)
	return json.Unmarshal(body, v)
}

var _ = middleware.GetTenant
