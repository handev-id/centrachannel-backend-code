package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"centrachannel/internal/src/tenant"
)

func TestAuthMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		c.Locals("tenant", &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Test Tenant"})
		c.Locals("user_id", uuid.MustParse("00000000-0000-4000-8000-000000000001"))
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

func DecodeData(resp *http.Response, v interface{}) error {
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return err
	}
	return json.Unmarshal(envelope.Data, v)
}
