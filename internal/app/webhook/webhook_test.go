package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/logger"
)

func TestProcessEvolutionEvent_UnknownEvent(t *testing.T) {
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "unknown"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProcessEvolutionEvent_NilData(t *testing.T) {
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.upsert"})
	if err != nil {
		t.Fatalf("expected nil error for nil data, got: %v", err)
	}
}

func TestProcessEvolutionEvent_FromMe_Sync(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "Contact", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001e"), ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-000000000190"), nil
		},
	}

	svc := newTestService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo)

	text := "Outgoing from phone"
	data, _ := json.Marshal(EvolutionMessageUpsert{
		Key:         EvolutionMessageKey{RemoteJid: "5511999999999@s.whatsapp.net", FromMe: true, ID: "from_phone_001"},
		PushName:    "Test Contact",
		MessageType: "conversation",
		Message:     EvolutionMessage{Conversation: &text},
	})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{
		Event:    "messages.upsert",
		Instance: "instance_test",
		Data:     data,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleMessageUpsert_NewConversation(t *testing.T) {
	var createdContact *contact.Contact
	var createdProfile *profile.Profile
	var createdConv *conversation.Conversation
	var createdMsg *message.Message

	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
			return nil, sql.ErrNoRows
		},
		createFunc: func(ctx context.Context, q contact.DBTX, c *contact.Contact) (uuid.UUID, error) {
			createdContact = c
			return uuid.MustParse("00000000-0000-4000-8000-000000000064"), nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
			return nil, sql.ErrNoRows
		},
		createFunc: func(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error) {
			createdProfile = p
			return uuid.MustParse("00000000-0000-4000-8000-0000000000c8"), nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Type: "whatsapp", Name: "WhatsApp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
			return nil, fmt.Errorf("not found")
		},
		createFunc: func(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (uuid.UUID, error) {
			createdConv = conv
			return uuid.MustParse("00000000-0000-4000-8000-00000000012c"), nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
			createdMsg = msg
			return uuid.MustParse("00000000-0000-4000-8000-000000000190"), nil
		},
	}

	svc := newTestService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo)

	text := "Hello, this is a test"
	payload := EvolutionWebhookPayload{
		Event:    "messages.upsert",
		Instance: "instance_test",
	}
	payloadData, _ := json.Marshal(EvolutionMessageUpsert{
		Key:         EvolutionMessageKey{RemoteJid: "5511999999999@s.whatsapp.net", FromMe: false, ID: "msg_001"},
		PushName:    "Test User",
		MessageType: "conversation",
		Message:     EvolutionMessage{Conversation: &text},
	})
	payload.Data = payloadData

	err := svc.ProcessEvolutionEvent(context.Background(), &payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdContact == nil {
		t.Fatal("expected contact to be created")
	}
	if createdProfile == nil {
		t.Fatal("expected profile to be created")
	}
	if createdConv == nil {
		t.Fatal("expected conversation to be created")
	}
	if createdMsg == nil {
		t.Fatal("expected message to be created")
	}
	if createdMsg.Text == nil || *createdMsg.Text != "Hello, this is a test" {
		t.Errorf("expected text 'Hello, this is a test', got %v", createdMsg.Text)
	}
	if createdMsg.Status != "unread" {
		t.Errorf("expected status 'unread', got %s", createdMsg.Status)
	}
}

func TestHandleMessageUpsert_ExistingContactProfileConversation(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "Existing", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001e"), ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-000000000190"), nil
		},
	}

	svc := newTestService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo)

	text := "Hello"
	data, _ := json.Marshal(EvolutionMessageUpsert{
		Key:         EvolutionMessageKey{RemoteJid: "5511999999999@s.whatsapp.net", FromMe: false, ID: "msg_002"},
		PushName:    "Existing User",
		MessageType: "conversation",
		Message:     EvolutionMessage{Conversation: &text},
	})

	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{
		Event: "messages.upsert", Instance: "instance_test", Data: data,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandleMessageUpsert_NotifiesNewMessage(t *testing.T) {
	notifier := &mockNotifier{}

	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "Existing", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001e"), ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-000000000190"), nil
		},
	}

	svc := newTestServiceWithNotifier(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, notifier)

	text := "Incoming whatsapp message"
	data, _ := json.Marshal(EvolutionMessageUpsert{
		Key:         EvolutionMessageKey{RemoteJid: "5511999999999@s.whatsapp.net", FromMe: false, ID: "msg_notify_001"},
		PushName:    "Existing User",
		MessageType: "conversation",
		Message:     EvolutionMessage{Conversation: &text},
	})

	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{
		Event: "messages.upsert", Instance: "instance_test", Data: data,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ev := notifier.last()
	if ev == nil {
		t.Fatal("expected a websocket notification")
	}
	if ev.tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected tenantID 1, got %s", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	msg, ok := ev.data.(*message.Message)
	if !ok {
		t.Fatalf("expected data *message.Message, got %T", ev.data)
	}
	if msg.ID != uuid.MustParse("00000000-0000-4000-8000-000000000190") {
		t.Errorf("expected message id 400, got %s", msg.ID)
	}
	if msg.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000001e") {
		t.Errorf("expected conversation_id 30, got %s", msg.ConversationID)
	}
	if msg.SenderType != "contact" {
		t.Errorf("expected sender_type contact, got %s", msg.SenderType)
	}
	if msg.Status != "unread" {
		t.Errorf("expected status unread, got %s", msg.Status)
	}
}

func TestProcessEvolutionEvent_FromMe_Sync_NotifiesNewMessage(t *testing.T) {
	notifier := &mockNotifier{}

	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "Contact", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001e"), ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-000000000190"), nil
		},
	}

	svc := newTestServiceWithNotifier(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, notifier)

	text := "Outgoing from phone"
	data, _ := json.Marshal(EvolutionMessageUpsert{
		Key:         EvolutionMessageKey{RemoteJid: "5511999999999@s.whatsapp.net", FromMe: true, ID: "from_phone_notify_001"},
		PushName:    "Test Contact",
		MessageType: "conversation",
		Message:     EvolutionMessage{Conversation: &text},
	})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{
		Event:    "messages.upsert",
		Instance: "instance_test",
		Data:     data,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ev := notifier.last()
	if ev == nil {
		t.Fatal("expected a websocket notification")
	}
	if ev.tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected tenantID 1, got %s", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	msg, ok := ev.data.(*message.Message)
	if !ok {
		t.Fatalf("expected data *message.Message, got %T", ev.data)
	}
	if msg.ID != uuid.MustParse("00000000-0000-4000-8000-000000000190") {
		t.Errorf("expected message id 400, got %s", msg.ID)
	}
	if msg.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000001e") {
		t.Errorf("expected conversation_id 30, got %s", msg.ConversationID)
	}
	if msg.SenderType != "user" {
		t.Errorf("expected sender_type user, got %s", msg.SenderType)
	}
}

func TestHandleMessageUpdate_Delivered(t *testing.T) {
	var capturedID, capturedStatus string
	var capturedTenant uuid.UUID
	tenantID := uuid.New()
	msgRepo := &mockMsgRepo{
		updateStatusByWebhookIDFunc: func(ctx context.Context, q message.DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
			capturedTenant = tenantID
			capturedID = webhookMessageID
			capturedStatus = status
			return nil
		},
	}
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, instance string) (*whatsapp_device.WhatsAppDevice, error) {
			if instance != "instance_test" {
				t.Fatalf("unexpected instance: %s", instance)
			}
			return &whatsapp_device.WhatsAppDevice{TenantID: tenantID}, nil
		},
	}
	svc := newTestService(deviceRepo, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)

	data, _ := json.Marshal(EvolutionMessageUpdate{
		KeyID: "webhook_001", Status: "DELIVERED",
	})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{
		Event: "messages.update", Instance: "instance_test", Data: data,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedID != "webhook_001" {
		t.Errorf("expected webhook_001, got %s", capturedID)
	}
	if capturedStatus != "delivered" {
		t.Errorf("expected 'delivered', got %s", capturedStatus)
	}
	if capturedTenant != tenantID {
		t.Errorf("status update lost device tenant: got %s, want %s", capturedTenant, tenantID)
	}
}

func TestHandleMessageUpdate_Read_Failed_Sent(t *testing.T) {
	tests := []struct{ status, expected string }{
		{"READ", "read"}, {"FAILED", "failed"}, {"SENT", "sent"}, {"DELIVERY_ACK", "delivered"}, {"PLAYED", "read"},
	}
	for _, tt := range tests {
		var capturedStatus string
		msgRepo := &mockMsgRepo{
			updateStatusByWebhookIDFunc: func(ctx context.Context, q message.DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
				capturedStatus = status
				return nil
			},
		}
		deviceRepo := &mockDeviceRepo{
			getByWhatsappIDFunc: func(context.Context, whatsapp_device.DBTX, string) (*whatsapp_device.WhatsAppDevice, error) {
				return &whatsapp_device.WhatsAppDevice{TenantID: uuid.New()}, nil
			},
		}
		svc := newTestService(deviceRepo, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)
		data, _ := json.Marshal(EvolutionMessageUpdate{
			KeyID: "w_" + tt.status, Status: tt.status,
		})
		svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "test", Data: data})
		if capturedStatus != tt.expected {
			t.Errorf("for %s: expected %s, got %s", tt.status, tt.expected, capturedStatus)
		}
	}
}

func TestHandleMessageUpdate_UnhandledStatus(t *testing.T) {
	called := false
	msgRepo := &mockMsgRepo{
		updateStatusByWebhookIDFunc: func(ctx context.Context, q message.DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
			called = true
			return nil
		},
	}
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)
	data, _ := json.Marshal(EvolutionMessageUpdate{
		KeyID: "w_004", Status: "PENDING",
	})
	svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "test", Data: data})
	if called {
		t.Error("expected no call for unhandled status")
	}
}

func TestHandleMessageUpdate_NilData(t *testing.T) {
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update"})
	if err != nil {
		t.Fatalf("expected nil error for nil data, got: %v", err)
	}
}

func TestHandleConnectionUpdate_Open(t *testing.T) {
	var updatedDevice *whatsapp_device.WhatsAppDevice
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WhatsappID: "instance_test", Status: "DISCONNECTED"}, nil
		},
		updateFunc: func(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID, device *whatsapp_device.WhatsAppDevice) error {
			updatedDevice = device
			return nil
		},
	}
	svc := newTestService(deviceRepo, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})

	data, _ := json.Marshal(EvolutionConnectionUpdate{Instance: "instance_test", State: "open"})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "connection.update", Instance: "instance_test", Data: data})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedDevice == nil || updatedDevice.Status != "CONNECTED" {
		t.Errorf("expected status CONNECTED, got %v", updatedDevice.Status)
	}
}

func TestHandleConnectionUpdate_Disconnected(t *testing.T) {
	var updatedDevice *whatsapp_device.WhatsAppDevice
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), WhatsappID: "instance_test", Status: "CONNECTED"}, nil
		},
		updateFunc: func(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID, device *whatsapp_device.WhatsAppDevice) error {
			updatedDevice = device
			return nil
		},
	}
	svc := newTestService(deviceRepo, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})

	data, _ := json.Marshal(EvolutionConnectionUpdate{Instance: "instance_test", State: "close"})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "connection.update", Instance: "instance_test", Data: data})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updatedDevice == nil || updatedDevice.Status != "DISCONNECTED" {
		t.Errorf("expected status DISCONNECTED, got %v", updatedDevice.Status)
	}
}

func TestHandleConnectionUpdate_NilData(t *testing.T) {
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "connection.update"})
	if err != nil {
		t.Fatalf("expected nil error for nil data, got: %v", err)
	}
}

func newMessageTestService(t *testing.T, capture func(*message.Message)) (*webhookService, *mockMsgRepo) {
	t.Helper()
	repo := &mockMsgRepo{createFunc: func(_ context.Context, _ message.DBTX, m *message.Message) (uuid.UUID, error) {
		capture(m)
		return uuid.New(), nil
	}}
	tenantID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	svc := newTestService(
		&mockDeviceRepo{getByWhatsappIDFunc: func(context.Context, whatsapp_device.DBTX, string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{TenantID: tenantID}, nil
		}},
		&mockContactRepo{getByPhoneFunc: func(context.Context, contact.DBTX, uuid.UUID, string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.New(), TenantID: tenantID, FirstName: "Customer"}, nil
		}},
		&mockProfileRepo{getByExternalIDAndChannelIDFunc: func(context.Context, profile.DBTX, uuid.UUID, string, uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000002")}, nil
		}},
		&mockChannelRepo{getByTypeFunc: func(context.Context, channel.DBTX, string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000003"), Type: "whatsapp"}, nil
		}},
		&mockConvRepo{findOpenByProfileAndChannelFunc: func(context.Context, conversation.DBTX, uuid.UUID, uuid.UUID, uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-000000000004")}, nil
		}},
		repo,
	)
	return svc.(*webhookService), repo
}

func TestIncomingQuoteKeepsOriginalContext(t *testing.T) {
	var created *message.Message
	svc, _ := newMessageTestService(t, func(m *message.Message) { created = m })
	raw := json.RawMessage(`{"key":{"id":"reply-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"messageType":"extendedTextMessage","message":{"extendedTextMessage":{"text":"Reply","contextInfo":{"stanzaId":"original-id","participant":"628123456789@s.whatsapp.net","quotedMessage":{"conversation":"Original"}}}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.upsert", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if created == nil {
		t.Fatal("no reply message created")
	}
	wire, _ := json.Marshal(created)
	var response struct {
		Reply *struct {
			ProviderMessageID string `json:"provider_message_id"`
			Text              string `json:"text"`
		} `json:"reply"`
	}
	if err := json.Unmarshal(wire, &response); err != nil {
		t.Fatal(err)
	}
	if response.Reply == nil || response.Reply.ProviderMessageID != "original-id" || response.Reply.Text != "Original" {
		t.Fatalf("quote context lost: %s", wire)
	}
}

func TestReactionUpsertDoesNotCreateBubble(t *testing.T) {
	created := 0
	svc, repo := newMessageTestService(t, func(*message.Message) { created++ })
	var mutation *message.Mutation
	repo.applyMutationFunc = func(_ context.Context, _ *sql.DB, _, _, _ uuid.UUID, event message.Mutation) (*message.Message, error) {
		mutation = &event
		return &message.Message{ID: uuid.New()}, nil
	}
	raw := json.RawMessage(`{"key":{"id":"reaction-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"messageType":"reactionMessage","message":{"reactionMessage":{"key":{"id":"target-id","fromMe":true},"text":"👍","senderTimestampMs":"1791894000123"}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.upsert", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if created != 0 || mutation == nil || mutation.TargetProviderID != "target-id" || mutation.Emoji != "👍" {
		t.Fatalf("reaction became a bubble: created=%d mutation=%+v", created, mutation)
	}
}

func TestEditedUpdateDoesNotBecomeDeliveryStatus(t *testing.T) {
	svc, repo := newMessageTestService(t, func(*message.Message) { t.Fatal("edit created a bubble") })
	var mutation *message.Mutation
	statusUpdates := 0
	repo.applyMutationFunc = func(_ context.Context, _ *sql.DB, _, _, _ uuid.UUID, event message.Mutation) (*message.Message, error) {
		mutation = &event
		return &message.Message{ID: uuid.New()}, nil
	}
	repo.updateStatusByWebhookIDFunc = func(context.Context, message.DBTX, uuid.UUID, string, string) error { statusUpdates++; return nil }
	raw := json.RawMessage(`{"keyId":"target-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false,"status":"SERVER_ACK","messageTimestamp":1791894001,"message":{"editedMessage":{"message":{"conversation":"Edited"}}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if statusUpdates != 0 || mutation == nil || mutation.Text == nil || *mutation.Text != "Edited" {
		t.Fatalf("edit routed as status: statuses=%d mutation=%+v", statusUpdates, mutation)
	}
}

func TestLegacyEditedWebhookUpdatesTarget(t *testing.T) {
	svc, repo := newMessageTestService(t, func(*message.Message) { t.Fatal("edit created a bubble") })
	var mutation *message.Mutation
	repo.applyMutationFunc = func(_ context.Context, _ *sql.DB, _, _, _ uuid.UUID, event message.Mutation) (*message.Message, error) {
		mutation = &event
		return &message.Message{ID: uuid.New()}, nil
	}
	raw := json.RawMessage(`{"key":{"id":"target-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"type":14,"editedMessage":{"conversation":"Legacy edit"},"timestampMs":"1791894000123"}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.edited", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if mutation == nil || mutation.Text == nil || *mutation.Text != "Legacy edit" {
		t.Fatalf("legacy edit lost: %+v", mutation)
	}
}

func TestIncomingQuoteResolvesStoredContactAndMedia(t *testing.T) {
	var created *message.Message
	svc, repo := newMessageTestService(t, func(m *message.Message) { created = m })
	targetID := uuid.New()
	repo.getByProviderIDFunc = func(_ context.Context, _ message.DBTX, tenant, profileID, channelID uuid.UUID, id string) (*message.Message, error) {
		if tenant != uuid.MustParse("00000000-0000-4000-8000-000000000001") || profileID != uuid.MustParse("00000000-0000-4000-8000-000000000002") || channelID != uuid.MustParse("00000000-0000-4000-8000-000000000003") || id != "original-id" {
			t.Fatal("quote lookup is not scoped to the peer")
		}
		return &message.Message{ID: targetID, SenderType: "contact", Attachment: json.RawMessage(`{"type":"image","url":"https://storage.example/photo.webp","status":"ready","caption":"Current caption"}`)}, nil
	}
	raw := json.RawMessage(`{"key":{"id":"reply-id","remoteJid":"628123456789@s.whatsapp.net","fromMe":false},"messageType":"extendedTextMessage","message":{"extendedTextMessage":{"text":"Reply","contextInfo":{"stanzaId":"original-id","participant":"123456789@lid","quotedMessage":{"imageMessage":{"url":"https://mmg.whatsapp.net/encrypted.enc","caption":"Snapshot caption","mimetype":"image/webp"}}}}}}`)
	if err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.upsert", Instance: "test", Data: raw}); err != nil {
		t.Fatal(err)
	}
	if created == nil || created.Reply == nil || created.Reply.SenderType != "contact" || created.Reply.MessageID == nil || *created.Reply.MessageID != targetID || created.Reply.SenderName != "Customer" {
		t.Fatalf("LID quote author not resolved: %+v", created)
	}
	var att message.Attachment
	if err := json.Unmarshal(created.Reply.Attachment, &att); err != nil {
		t.Fatal(err)
	}
	if att.URL != "https://storage.example/photo.webp" || att.Caption != "Snapshot caption" {
		t.Fatalf("quote lost stored thumbnail or snapshot caption: %s", created.Reply.Attachment)
	}
}

type mockWebhookService struct {
	processEvolutionFunc func(ctx context.Context, payload *EvolutionWebhookPayload) error
	processMetaFunc      func(ctx context.Context, payload *MetaWebhookPayload) error
}

func (m *mockWebhookService) ProcessEvolutionEvent(ctx context.Context, payload *EvolutionWebhookPayload) error {
	if m.processEvolutionFunc != nil {
		return m.processEvolutionFunc(ctx, payload)
	}
	return nil
}

func (m *mockWebhookService) ProcessMetaEvent(ctx context.Context, payload *MetaWebhookPayload) error {
	if m.processMetaFunc != nil {
		return m.processMetaFunc(ctx, payload)
	}
	return nil
}

func setupApp(apiKey, metaSecret string) *fiber.App {
	app := fiber.New()
	handler := &WebhookHandler{service: &mockWebhookService{}, apiKey: apiKey, metaSecret: metaSecret, metaAppSecret: "test-meta-app-secret"}

	app.Post("/webhook/evolution", handler.HandleEvolution)
	app.Get("/webhook/meta", handler.HandleMetaVerify)
	app.Post("/webhook/meta", handler.HandleMetaWebhook)

	return app
}

func TestHandleEvolution_NoAuth(t *testing.T) {
	app := setupApp("secret123", "")

	body := bytes.NewReader([]byte(`{"event":"connection.update"}`))
	req := httptest.NewRequest(http.MethodPost, "/webhook/evolution", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandleEvolution_BodyAPIKey(t *testing.T) {
	app := setupApp("secret123", "")

	body := bytes.NewReader([]byte(`{"event":"connection.update","apikey":"secret123"}`))
	req := httptest.NewRequest(http.MethodPost, "/webhook/evolution", body)
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleMetaVerify_Success(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=subscribe&hub.verify_token=meta_secret_456&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	if buf.String() != "challenge_abc" {
		t.Errorf("expected 'challenge_abc', got %s", buf.String())
	}
}

func TestHandleMetaVerify_WrongToken(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=subscribe&hub.verify_token=wrong&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestHandleMetaVerify_WrongMode(t *testing.T) {
	app := setupApp("", "meta_secret_456")

	req := httptest.NewRequest(http.MethodGet, "/webhook/meta?hub.mode=unsubscribe&hub.verify_token=meta_secret_456&hub.challenge=challenge_abc", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}

func TestHandleMetaWebhook_Success(t *testing.T) {
	app := setupApp("", "")

	payload := MetaWebhookPayload{
		Object: "page",
		Entry: []MetaWebhookEntry{{
			ID: "page_123",
			Messaging: []MetaWebhookMessage{{
				Sender:  &MetaSender{ID: "sender_456"},
				Message: &MetaMessage{MID: "mid_789", Text: "Hello"},
			}},
		}},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook/meta", bytes.NewReader(body))
	mac := hmac.New(sha256.New, []byte("test-meta-app-secret"))
	_, _ = mac.Write(body)
	req.Header.Set("x-hub-signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHandleMetaWebhook_Instagram(t *testing.T) {
	app := setupApp("", "")

	payload := MetaWebhookPayload{
		Object: "instagram",
		Entry: []MetaWebhookEntry{{
			ID: "ig_biz_123",
			Messaging: []MetaWebhookMessage{{
				Sender:  &MetaSender{ID: "ig_sender_456"},
				Message: &MetaMessage{MID: "ig_mid_789", Text: "Hello from IG"},
			}},
		}},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/webhook/meta", bytes.NewReader(body))
	mac := hmac.New(sha256.New, []byte("test-meta-app-secret"))
	_, _ = mac.Write(body)
	req.Header.Set("x-hub-signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestExtractMessageContent(t *testing.T) {
	// conversation
	text := "Hello"
	txt, att := extractMessageContent(EvolutionMessage{Conversation: &text}, "conversation")
	if txt == nil || *txt != "Hello" {
		t.Errorf("expected 'Hello', got %v", txt)
	}
	if att != nil {
		t.Error("expected nil attachment")
	}

	// extendedTextMessage
	msg := EvolutionMessage{ExtendedTextMessage: &struct {
		Text string `json:"text"`
	}{Text: "Extended"}}
	txt, att = extractMessageContent(msg, "extendedTextMessage")
	if txt == nil || *txt != "Extended" {
		t.Errorf("expected 'Extended', got %v", txt)
	}
	if att != nil {
		t.Error("expected nil attachment")
	}

	// image with caption
	imgMsg := EvolutionMessage{ImageMessage: &struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	}{URL: "https://img.url/p.jpg", Mimetype: "image/jpeg", Caption: "Nice pic"}}
	txt, att = extractMessageContent(imgMsg, "imageMessage")
	if txt == nil || *txt != "Nice pic" {
		t.Errorf("expected 'Nice pic', got %v", txt)
	}
	if att == nil {
		t.Fatal("expected attachment")
	}

	// image without caption
	imgNoCap := EvolutionMessage{ImageMessage: &struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	}{URL: "https://img.url/p.jpg", Mimetype: "image/jpeg"}}
	txt, att = extractMessageContent(imgNoCap, "imageMessage")
	if txt != nil {
		t.Errorf("expected nil text, got %v", *txt)
	}
	if att == nil {
		t.Fatal("expected attachment")
	}

	// video
	vidMsg := EvolutionMessage{VideoMessage: &struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	}{URL: "https://vid.url/v.mp4", Mimetype: "video/mp4", Caption: "Check this"}}
	txt, att = extractMessageContent(vidMsg, "videoMessage")
	if txt == nil || *txt != "Check this" {
		t.Errorf("expected 'Check this', got %v", txt)
	}
	if att == nil {
		t.Fatal("expected attachment")
	}

	// audio
	audMsg := EvolutionMessage{AudioMessage: &struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Base64   string `json:"base64,omitempty"`
	}{URL: "https://audio.url/v.ogg", Mimetype: "audio/ogg"}}
	txt, att = extractMessageContent(audMsg, "audioMessage")
	if txt != nil {
		t.Errorf("expected nil text, got %v", *txt)
	}
	if att == nil {
		t.Fatal("expected attachment")
	}

	// document
	docMsg := EvolutionMessage{DocumentMessage: &struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		FileName string `json:"fileName,omitempty"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	}{URL: "https://doc.url/r.pdf", Mimetype: "application/pdf", FileName: "report.pdf"}}
	txt, att = extractMessageContent(docMsg, "documentMessage")
	if txt != nil {
		t.Errorf("expected nil text, got %v", *txt)
	}
	if att == nil {
		t.Fatal("expected attachment")
	}

	// empty
	txt, att = extractMessageContent(EvolutionMessage{}, "")
	if txt != nil || att != nil {
		t.Error("expected nil text and nil attachment")
	}
}

func TestExtractMessageContent_Base64Fallback(t *testing.T) {
	// image with base64 but no valid URL (CDN path)
	imgMsg := EvolutionMessage{ImageMessage: &struct {
		URL      string `json:"url"`
		Mimetype string `json:"mimetype"`
		Caption  string `json:"caption,omitempty"`
		Base64   string `json:"base64,omitempty"`
	}{URL: "/v/t62.11036-24/1198/HASH/file.jpg", Mimetype: "image/jpeg", Base64: "AAAAHGZ0eXBpc29"}}
	txt, att := extractMessageContent(imgMsg, "imageMessage")
	if txt != nil {
		t.Errorf("expected nil text, got %v", *txt)
	}
	if att == nil {
		t.Fatal("expected attachment")
	}

	// verify base64 is included
	var attMap map[string]interface{}
	if err := json.Unmarshal(att, &attMap); err != nil {
		t.Fatalf("failed to unmarshal attachment: %v", err)
	}
	if attMap["base64"] != "AAAAHGZ0eXBpc29" {
		t.Errorf("expected base64 data, got %v", attMap["base64"])
	}
	url, ok := attMap["url"].(string)
	if !ok || url == "" {
		t.Error("expected url to be set from base64 data URL")
	}
}

func TestExtractMessageContent_AnimatedSticker(t *testing.T) {
	var incoming EvolutionMessage
	if err := json.Unmarshal([]byte(`{"stickerMessage":{"url":"https://cdn.example/sticker.enc","mimetype":"image/webp","isAnimated":true},"base64":"d2VicA=="}`), &incoming); err != nil {
		t.Fatal(err)
	}
	text, raw := extractMessageContent(incoming, "stickerMessage")
	var att map[string]interface{}
	if err := json.Unmarshal(raw, &att); err != nil {
		t.Fatal(err)
	}
	if text != nil || att["type"] != "sticker" || att["is_animated"] != true || att["mimetype"] != "image/webp" || att["base64"] != "d2VicA==" {
		t.Fatalf("expected playable animated sticker metadata, got text=%v attachment=%s", text, raw)
	}
}

func TestProcessMetaEvent_NotifiesNewMessage(t *testing.T) {
	notifier := &mockNotifier{}

	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), FirstName: "Existing", Facebook: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Type: channelType}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001e"), ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-0000000001f4"), nil
		},
	}

	svc := newTestServiceWithNotifier(&mockDeviceRepo{}, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, notifier)

	payload := &MetaWebhookPayload{
		Object: "page",
		Entry: []MetaWebhookEntry{
			{
				ID:   "page_123",
				Time: 1700000000,
				Messaging: []MetaWebhookMessage{
					{
						Sender:  &MetaSender{ID: "fb_sender_1"},
						Message: &MetaMessage{MID: "mid_001", Text: "Hello from facebook"},
					},
				},
			},
		},
	}

	err := svc.ProcessMetaEvent(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ev := notifier.last()
	if ev == nil {
		t.Fatal("expected a websocket notification")
	}
	if ev.tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected tenantID 1, got %s", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	msg, ok := ev.data.(*message.Message)
	if !ok {
		t.Fatalf("expected data *message.Message, got %T", ev.data)
	}
	if msg.ID != uuid.MustParse("00000000-0000-4000-8000-0000000001f4") {
		t.Errorf("expected message id 500, got %s", msg.ID)
	}
	if msg.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000001e") {
		t.Errorf("expected conversation_id 30, got %s", msg.ConversationID)
	}
	if msg.SenderType != "contact" {
		t.Errorf("expected sender_type contact, got %s", msg.SenderType)
	}
	if msg.Text == nil || *msg.Text != "Hello from facebook" {
		t.Errorf("expected text 'Hello from facebook', got %v", msg.Text)
	}
}

func TestHandleMessageUpdate_UnknownInstanceCannotMutate(t *testing.T) {
	called := false
	msgRepo := &mockMsgRepo{
		updateStatusByWebhookIDFunc: func(context.Context, message.DBTX, uuid.UUID, string, string) error {
			called = true
			return nil
		},
	}
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)
	data, err := json.Marshal(EvolutionMessageUpdate{KeyID: "shared-provider-id", Status: "DELIVERED"})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "unknown-instance", Data: data})
	if err == nil {
		t.Error("unknown instance must be rejected")
	}
	if called {
		t.Error("unknown instance reached message update")
	}
}

// ---- mock repositories ----

type mockDeviceRepo struct {
	getByWhatsappIDFunc func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error)
	updateFunc          func(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID, device *whatsapp_device.WhatsAppDevice) error
}

func (m *mockDeviceRepo) List(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID) ([]whatsapp_device.WhatsAppDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) GetByID(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID) (*whatsapp_device.WhatsAppDevice, error) {
	return nil, nil
}
func (m *mockDeviceRepo) GetByWhatsappID(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
	if m.getByWhatsappIDFunc != nil {
		return m.getByWhatsappIDFunc(ctx, q, whatsappID)
	}
	return nil, nil
}
func (m *mockDeviceRepo) Create(ctx context.Context, q whatsapp_device.DBTX, device *whatsapp_device.WhatsAppDevice) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockDeviceRepo) Update(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID, device *whatsapp_device.WhatsAppDevice) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, q, tenantID, id, device)
	}
	return nil
}
func (m *mockDeviceRepo) Delete(ctx context.Context, q whatsapp_device.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}

type mockContactRepo struct {
	contact.ContactRepository
	getByPhoneFunc func(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error)
	createFunc     func(ctx context.Context, q contact.DBTX, c *contact.Contact) (uuid.UUID, error)
}

func (m *mockContactRepo) List(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, limit, offset int, f contact.ListContactQuery) ([]*contact.Contact, int, error) {
	return nil, 0, nil
}
func (m *mockContactRepo) GetByID(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, id uuid.UUID) (*contact.Contact, error) {
	return nil, nil
}
func (m *mockContactRepo) GetByPhone(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, phone string) (*contact.Contact, error) {
	if m.getByPhoneFunc != nil {
		return m.getByPhoneFunc(ctx, q, tenantID, phone)
	}
	return nil, nil
}
func (m *mockContactRepo) Create(ctx context.Context, q contact.DBTX, c *contact.Contact) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, c)
	}
	return uuid.Nil, nil
}
func (m *mockContactRepo) Update(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, id uuid.UUID, c *contact.Contact) error {
	return nil
}
func (m *mockContactRepo) SoftDelete(ctx context.Context, q contact.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}

type mockProfileRepo struct {
	getByExternalIDAndChannelIDFunc func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error)
	createFunc                      func(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error)
}

func (m *mockProfileRepo) List(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID, channelID uuid.UUID) ([]profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID) (*profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByExternalIDAndChannelID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
	if m.getByExternalIDAndChannelIDFunc != nil {
		return m.getByExternalIDAndChannelIDFunc(ctx, q, tenantID, externalID, channelID)
	}
	return nil, nil
}
func (m *mockProfileRepo) Create(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, p)
	}
	return uuid.Nil, nil
}
func (m *mockProfileRepo) Update(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID, p *profile.Profile) error {
	return nil
}
func (m *mockProfileRepo) GetByContactID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID uuid.UUID) ([]profile.Profile, error) {
	return nil, nil
}
func (m *mockProfileRepo) GetByContactIDs(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID][]profile.Profile, error) {
	return map[uuid.UUID][]profile.Profile{}, nil
}

type mockChannelRepo struct {
	getByTypeFunc func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error)
}

func (m *mockChannelRepo) GetByID(ctx context.Context, q channel.DBTX, id uuid.UUID) (*channel.Channel, error) {
	return &channel.Channel{}, nil
}
func (m *mockChannelRepo) GetByType(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
	if m.getByTypeFunc != nil {
		return m.getByTypeFunc(ctx, q, channelType)
	}
	return &channel.Channel{}, nil
}
func (m *mockChannelRepo) List(ctx context.Context, q channel.DBTX) ([]channel.Channel, error) {
	return nil, nil
}

type mockConvRepo struct {
	createFunc                      func(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (uuid.UUID, error)
	updateLastMessageFunc           func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error
	findOpenByProfileAndChannelFunc func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error)
}

func (m *mockConvRepo) GetByID(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) (*conversation.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) Create(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, conv)
	}
	return uuid.Nil, nil
}
func (m *mockConvRepo) UpdateStatus(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	return nil
}
func (m *mockConvRepo) Assign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, agentID uuid.UUID) error {
	return nil
}
func (m *mockConvRepo) Unassign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}
func (m *mockConvRepo) GetTotalUnread(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockConvRepo) MarkRead(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	return nil
}
func (m *mockConvRepo) ListCursor(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, limit int, status string, channelID, agentID uuid.UUID, search string, lastActivity *time.Time, lastID uuid.UUID) ([]*conversation.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) UpdateLastMessage(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
	if m.updateLastMessageFunc != nil {
		return m.updateLastMessageFunc(ctx, q, tenantID, id, lastMessageJSON, lastAgentID)
	}
	return nil
}
func (m *mockConvRepo) FindOpenByProfileAndChannel(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
	if m.findOpenByProfileAndChannelFunc != nil {
		return m.findOpenByProfileAndChannelFunc(ctx, q, tenantID, profileID, channelID)
	}
	return nil, fmt.Errorf("not found")
}

type mockMsgRepo struct {
	getByProviderIDFunc func(context.Context, message.DBTX, uuid.UUID, uuid.UUID, uuid.UUID, string) (*message.Message, error)
	applyMutationFunc   func(context.Context, *sql.DB, uuid.UUID, uuid.UUID, uuid.UUID, message.Mutation) (*message.Message, error)
	message.MessageRepository
	createFunc                  func(ctx context.Context, q message.DBTX, msg *message.Message) (uuid.UUID, error)
	updateStatusByWebhookIDFunc func(ctx context.Context, q message.DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error
}

func (m *mockMsgRepo) GetByProviderID(ctx context.Context, q message.DBTX, tenantID, profileID, channelID uuid.UUID, providerID string) (*message.Message, error) {
	if m.getByProviderIDFunc != nil {
		return m.getByProviderIDFunc(ctx, q, tenantID, profileID, channelID, providerID)
	}
	return nil, message.ErrMessageNotFound
}

func (m *mockMsgRepo) ApplyMutation(ctx context.Context, db *sql.DB, tenantID, profileID, channelID uuid.UUID, event message.Mutation) (*message.Message, error) {
	if m.applyMutationFunc != nil {
		return m.applyMutationFunc(ctx, db, tenantID, profileID, channelID, event)
	}
	return nil, nil
}

func (m *mockMsgRepo) Create(ctx context.Context, db *sql.DB, msg *message.Message, lastMessageJSON []byte, lastAgentID *uuid.UUID, conversations conversation.ConversationRepository) (uuid.UUID, error) {
	if conversations != nil {
		if err := conversations.UpdateLastMessage(ctx, db, msg.TenantID, msg.ConversationID, lastMessageJSON, lastAgentID); err != nil {
			return uuid.Nil, err
		}
	}
	if m.createFunc != nil {
		return m.createFunc(ctx, db, msg)
	}
	return uuid.Nil, nil
}
func (m *mockMsgRepo) UpdateStatus(ctx context.Context, q message.DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	return nil
}
func (m *mockMsgRepo) UpdateStatusByWebhookID(ctx context.Context, q message.DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
	if m.updateStatusByWebhookIDFunc != nil {
		return m.updateStatusByWebhookIDFunc(ctx, q, tenantID, webhookMessageID, status)
	}
	return nil
}
func (m *mockMsgRepo) ListCursor(ctx context.Context, q message.DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*message.Message, error) {
	return nil, nil
}
func (m *mockMsgRepo) UpdateWebhookID(ctx context.Context, db *sql.DB, tenantID uuid.UUID, id uuid.UUID, webhookMessageID string) error {
	return nil
}
func (m *mockMsgRepo) MarkReadByConversation(ctx context.Context, q message.DBTX, tenantID uuid.UUID, conversationID uuid.UUID) error {
	return nil
}
func (m *mockMsgRepo) GetUnreadByConversation(ctx context.Context, q message.DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]*message.Message, error) {
	return nil, nil
}

// ---- test helpers ----

type notifiedEvent struct {
	tenantID uuid.UUID
	event    string
	data     interface{}
}

type mockNotifier struct {
	events []notifiedEvent
}

func (m *mockNotifier) Notify(tenantID uuid.UUID, event string, data interface{}) {
	m.events = append(m.events, notifiedEvent{tenantID: tenantID, event: event, data: data})
}

func (m *mockNotifier) last() *notifiedEvent {
	if len(m.events) == 0 {
		return nil
	}
	return &m.events[len(m.events)-1]
}

type mockMetaTenantRepo struct{}

func (m *mockMetaTenantRepo) Create(ctx context.Context, q tenant.DBTX, t *tenant.Tenant) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockMetaTenantRepo) CreateRole(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockMetaTenantRepo) CreateUser(ctx context.Context, q tenant.DBTX, u *tenant.User) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockMetaTenantRepo) AttachRole(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error {
	return nil
}
func (m *mockMetaTenantRepo) List(ctx context.Context, q tenant.DBTX) ([]tenant.Tenant, error) {
	return nil, nil
}
func (m *mockMetaTenantRepo) GetByID(ctx context.Context, q tenant.DBTX, id uuid.UUID) (*tenant.Tenant, error) {
	return nil, nil
}
func (m *mockMetaTenantRepo) GetByMetaPageID(ctx context.Context, q tenant.DBTX, pageID string) (*tenant.Tenant, error) {
	return &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Test Tenant"}, nil
}
func (m *mockMetaTenantRepo) GetByMetaInstagramBusinessID(ctx context.Context, q tenant.DBTX, igID string) (*tenant.Tenant, error) {
	return &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Test Tenant"}, nil
}
func (m *mockMetaTenantRepo) Update(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) error {
	return nil
}

func newTestService(deviceRepo whatsapp_device.WhatsAppDeviceRepository, contactRepo contact.ContactRepository, profileRepo profile.ProfileRepository, channelRepo channel.ChannelRepository, convRepo conversation.ConversationRepository, msgRepo message.MessageRepository) WebhookService {
	return newTestServiceWithNotifier(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, &mockNotifier{})
}

func newTestServiceWithNotifier(deviceRepo whatsapp_device.WhatsAppDeviceRepository, contactRepo contact.ContactRepository, profileRepo profile.ProfileRepository, channelRepo channel.ChannelRepository, convRepo conversation.ConversationRepository, msgRepo message.MessageRepository, notifier *mockNotifier) *webhookService {
	log := logger.NewLogger("debug", "text")
	tenants := &mockMetaTenantRepo{}
	messages := message.NewMessageService(msgRepo, convRepo, profileRepo, channelRepo, tenants, nil, nil, log, nil, notifier)
	return NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, messages, tenants, nil, log, nil, notifier)
}

func (m *mockMetaTenantRepo) GetDomains(ctx context.Context, q tenant.DBTX, id uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage("[]"), nil
}
func (m *mockMetaTenantRepo) GetActiveIDByDomain(ctx context.Context, q tenant.DBTX, name string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockMsgRepo) RetryFailed(ctx context.Context, q message.DBTX, tenantID uuid.UUID, id uuid.UUID, senderID uuid.UUID) (*message.Message, error) {
	return nil, fmt.Errorf("not found")
}
