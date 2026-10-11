package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
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

	svc := NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, &mockMetaTenantRepo{}, nil, logger.NewLogger("debug", "text"), nil, notifier)

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

	svc := NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, &mockMetaTenantRepo{}, nil, logger.NewLogger("debug", "text"), nil, notifier)

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
