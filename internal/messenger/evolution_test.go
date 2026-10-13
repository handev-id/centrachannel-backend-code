package messenger

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"centrachannel/internal/utils/logger"
)

var testLogger = logger.NewLogger("error", "text")

func TestNewEvolutionSender(t *testing.T) {
	s := NewEvolutionSender(EvolutionConfig{APIURL: "http://example.com", APIKey: "key123", DeviceID: "dev1"}, testLogger)
	if s == nil {
		t.Fatal("expected non-nil sender")
	}
	if s.cfg.APIKey != "key123" {
		t.Errorf("expected APIKey key123, got %s", s.cfg.APIKey)
	}
}

func TestSend_NoTextNoAttachment(t *testing.T) {
	s := NewEvolutionSender(EvolutionConfig{APIURL: "http://example.com", APIKey: "key123", DeviceID: "dev1"}, testLogger)
	_, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999"})
	if err == nil {
		t.Fatal("expected error for no text or attachment")
	}
}

func TestSend_Text_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apikey") != "key123" {
			t.Errorf("expected apikey key123, got %s", r.Header.Get("apikey"))
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","key":{"id":"msg123"}}`))
	}))
	defer server.Close()

	s := NewEvolutionSender(EvolutionConfig{APIURL: server.URL, APIKey: "key123", DeviceID: "dev1"}, testLogger)
	text := "Hello"
	msgID, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999", Text: &text})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "msg123" {
		t.Errorf("expected msg123, got %s", msgID)
	}
}

func TestSend_Text_EvolutionError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"error","error":"invalid number"}`))
	}))
	defer server.Close()

	s := NewEvolutionSender(EvolutionConfig{APIURL: server.URL, APIKey: "key123", DeviceID: "dev1"}, testLogger)
	text := "Hello"
	_, err := s.Send(&OutgoingMessage{RecipientID: `invalid`, Text: &text})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSend_Text_MissingMessageID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"sent","key":{"id":""}}`))
	}))
	defer server.Close()

	s := NewEvolutionSender(EvolutionConfig{APIURL: server.URL, APIKey: "key123", DeviceID: "dev1"}, testLogger)
	text := "Hello"
	msgID, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999", Text: &text})
	if err == nil {
		t.Fatalf("expected error for missing message ID, got %q", msgID)
	}
}

func TestSend_Text_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`not json`))
	}))
	defer server.Close()

	s := NewEvolutionSender(EvolutionConfig{APIURL: server.URL, APIKey: "key123", DeviceID: "dev1"}, testLogger)
	text := "Hello"
	_, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999", Text: &text})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSend_Attachment_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","key":{"id":"media456"}}`))
	}))
	defer server.Close()

	att := json.RawMessage(`{"url":"https://example.com/doc.pdf","type":"document","fileName":"report.pdf"}`)
	s := NewEvolutionSender(EvolutionConfig{APIURL: server.URL, APIKey: "key123", DeviceID: "dev1"}, testLogger)
	msgID, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999", Attachment: att})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "media456" {
		t.Errorf("expected media456, got %s", msgID)
	}
}

func TestSend_Attachment_InvalidAttachment(t *testing.T) {
	s := NewEvolutionSender(EvolutionConfig{APIURL: "http://example.com", APIKey: "key123", DeviceID: "dev1"}, testLogger)
	_, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999", Attachment: json.RawMessage(`invalid`)})
	if err == nil {
		t.Fatal("expected error for invalid attachment data")
	}
}

func TestSend_Attachment_EmptyURL(t *testing.T) {
	s := NewEvolutionSender(EvolutionConfig{APIURL: "http://example.com", APIKey: "key123", DeviceID: "dev1"}, testLogger)
	_, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999", Attachment: json.RawMessage(`{"url":""}`)})
	if err == nil {
		t.Fatal("expected error for empty url")
	}
}

func TestDoRequest_NetworkError(t *testing.T) {
	s := NewEvolutionSender(EvolutionConfig{APIURL: "http://invalid.local:12345", APIKey: "key123", DeviceID: "dev1"}, testLogger)
	_, err := s.doRequest("http://invalid.local:12345/send", map[string]interface{}{"text": "hello"})
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestSend_Attachment_MediaTypeDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"success","key":{"id":"default"}}`))
	}))
	defer server.Close()

	att := json.RawMessage(`{"url":"https://example.com/file","type":""}`)
	s := NewEvolutionSender(EvolutionConfig{APIURL: server.URL, APIKey: "key123", DeviceID: "dev1"}, testLogger)
	msgID, err := s.Send(&OutgoingMessage{RecipientID: "5511999999999", Attachment: att})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msgID != "default" {
		t.Errorf("expected default, got %s", msgID)
	}
}

func TestEvolutionReplyKeepsProviderQuote(t *testing.T) {
	for _, kind := range []string{"text", "media"} {
		t.Run(kind, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body struct {
					Quoted struct {
						Key struct {
							ID     string `json:"id"`
							FromMe bool   `json:"fromMe"`
						} `json:"key"`
						Message struct {
							Conversation string `json:"conversation"`
						} `json:"message"`
					} `json:"quoted"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if body.Quoted.Key.ID != "original-id" || !body.Quoted.Key.FromMe || body.Quoted.Message.Conversation != "Original" {
					t.Errorf("provider quote missing: %+v", body)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"key":{"id":"reply-id"}}`))
			}))
			defer server.Close()
			text := "Reply"
			msg := &OutgoingMessage{RecipientID: "628123456789", Text: &text}
			if kind == "media" {
				msg.Attachment = json.RawMessage(`{"type":"image","url":"https://storage.example/image.jpg"}`)
			}
			original := "Original"
			msg.Quoted = &QuotedMessage{ProviderMessageID: "original-id", FromMe: true, Text: &original}

			sender := NewEvolutionSender(EvolutionConfig{APIURL: server.URL, DeviceID: "test"}, testLogger)
			if _, err := sender.Send(msg); err != nil {
				t.Fatal(err)
			}
		})
	}
}
