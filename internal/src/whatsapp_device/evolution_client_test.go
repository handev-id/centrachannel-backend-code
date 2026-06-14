package whatsapp_device

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"centrachannel/internal/utils/logger"
)

var testDevice = &WhatsAppDevice{
	ID:         1,
	TenantID:   1,
	Name:       "Test Device",
	WhatsappID: "instance_test",
	Status:     "DISCONNECTED",
}

func TestNewEvolutionClient(t *testing.T) {
	c := NewEvolutionClient("http://example.com", "key123", logger.NewLogger("debug", "text"))
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestGetQR_Base64(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.Header.Get("apikey") != "key123" {
			t.Errorf("expected apikey key123")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"base64":"data:image/png;base64,qr123","code":""}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	qr, err := c.GetQR(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qr != "data:image/png;base64,qr123" {
		t.Errorf("expected base64 qr, got %s", qr)
	}
}

func TestGetQR_Code(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"base64":"","code":"ABCD1234"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	qr, err := c.GetQR(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if qr != "ABCD1234" {
		t.Errorf("expected ABCD1234, got %s", qr)
	}
}

func TestGetQR_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error":"instance not found"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	_, err := c.GetQR(context.Background(), testDevice)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetQR_NoQR(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"base64":"","code":""}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	_, err := c.GetQR(context.Background(), testDevice)
	if err == nil {
		t.Fatal("expected error for no qr")
	}
}

func TestCheckConnection_Open(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"instance":{"state":"open"}}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	connected, err := c.CheckConnection(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !connected {
		t.Error("expected connected true")
	}
}

func TestCheckConnection_Closed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"instance":{"state":"close"}}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	connected, err := c.CheckConnection(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if connected {
		t.Error("expected connected false")
	}
}

func TestCheckConnection_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error":"instance error"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	_, err := c.CheckConnection(context.Background(), testDevice)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateInstance_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.CreateInstance(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateInstance_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error":"already exists"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.CreateInstance(context.Background(), testDevice)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateInstance_InvalidJSONReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.CreateInstance(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("expected nil error for invalid json, got: %v", err)
	}
}

func TestDeleteInstance_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.DeleteInstance(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteInstance_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.DeleteInstance(context.Background(), testDevice)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeleteInstance_InvalidJSONReturnsNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.DeleteInstance(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("expected nil error for invalid json, got: %v", err)
	}
}

func TestSetWebhook_Success(t *testing.T) {
	var capturedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		capturedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.SetWebhook(context.Background(), testDevice, "https://webhook.test/evolution")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedURL == "" {
		t.Error("expected request to be made")
	}
}

func TestSetWebhook_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error":"invalid url"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.SetWebhook(context.Background(), testDevice, "bad")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDisconnect_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.Disconnect(context.Background(), testDevice)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDisconnect_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error":"not found"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.Disconnect(context.Background(), testDevice)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDisconnect_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	err := c.Disconnect(context.Background(), testDevice)
	if err == nil {
		t.Fatal("expected error for invalid json")
	}
}

func TestSendMessage_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","key":{"id":"evo_msg_123"}}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	result, err := c.SendMessage(context.Background(), testDevice, "5511999999999", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID != "evo_msg_123" {
		t.Errorf("expected evo_msg_123, got %s", result.MessageID)
	}
	if result.Status != "success" {
		t.Errorf("expected status success, got %s", result.Status)
	}
}

func TestSendMessage_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"error":"invalid number","status":"error"}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	_, err := c.SendMessage(context.Background(), testDevice, "invalid", "Hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSendMessage_EmptyMessageID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"sent","key":{"id":""}}`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	result, err := c.SendMessage(context.Background(), testDevice, "5511999999999", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.MessageID == "" {
		t.Error("expected non-empty fallback MessageID")
	}
}

func TestSendMessage_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	c := NewEvolutionClient(server.URL, "key123", logger.NewLogger("debug", "text"))
	_, err := c.SendMessage(context.Background(), testDevice, "5511999999999", "Hello")
	if err == nil {
		t.Fatal("expected error")
	}
}
