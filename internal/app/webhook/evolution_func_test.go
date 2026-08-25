package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

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
			return &whatsapp_device.WhatsAppDevice{ID: 1, TenantID: 1, WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: 10, TenantID: 1, FirstName: "Contact", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
			return &profile.Profile{ID: 20, ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: 1, Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, profileID int, channelID int) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: 30, ProfileID: 20, Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error { return nil },
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) { return 400, nil },
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
			return &whatsapp_device.WhatsAppDevice{ID: 1, TenantID: 1, WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
			return nil, sql.ErrNoRows
		},
		createFunc: func(ctx context.Context, q contact.DBTX, c *contact.Contact) (int, error) {
			createdContact = c
			return 100, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
			return nil, sql.ErrNoRows
		},
		createFunc: func(ctx context.Context, q profile.DBTX, p *profile.Profile) (int, error) {
			createdProfile = p
			return 200, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: 1, Type: "whatsapp", Name: "WhatsApp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, profileID int, channelID int) (*conversation.Conversation, error) {
			return nil, fmt.Errorf("not found")
		},
		createFunc: func(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (int, error) {
			createdConv = conv
			return 300, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) {
			createdMsg = msg
			return 400, nil
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
	if createdContact == nil { t.Fatal("expected contact to be created") }
	if createdProfile == nil { t.Fatal("expected profile to be created") }
	if createdConv == nil { t.Fatal("expected conversation to be created") }
	if createdMsg == nil { t.Fatal("expected message to be created") }
	if createdMsg.Text == nil || *createdMsg.Text != "Hello, this is a test" {
		t.Errorf("expected text 'Hello, this is a test', got %v", createdMsg.Text)
	}
	if createdMsg.Status != "unread" { t.Errorf("expected status 'unread', got %s", createdMsg.Status) }
}

func TestHandleMessageUpsert_ExistingContactProfileConversation(t *testing.T) {
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: 1, TenantID: 1, WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: 10, TenantID: 1, FirstName: "Existing", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
			return &profile.Profile{ID: 20, ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: 1, Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, profileID int, channelID int) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: 30, ProfileID: 20, Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error { return nil },
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) { return 400, nil },
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
			return &whatsapp_device.WhatsAppDevice{ID: 1, TenantID: 1, WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: 10, TenantID: 1, FirstName: "Existing", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
			return &profile.Profile{ID: 20, ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: 1, Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, profileID int, channelID int) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: 30, ProfileID: 20, Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) { return 400, nil },
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
	if ev.tenantID != 1 {
		t.Errorf("expected tenantID 1, got %d", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	msg, ok := ev.data.(*message.Message)
	if !ok {
		t.Fatalf("expected data *message.Message, got %T", ev.data)
	}
	if msg.ID != 400 {
		t.Errorf("expected message id 400, got %d", msg.ID)
	}
	if msg.ConversationID != 30 {
		t.Errorf("expected conversation_id 30, got %d", msg.ConversationID)
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
			return &whatsapp_device.WhatsAppDevice{ID: 1, TenantID: 1, WhatsappID: "instance_test"}, nil
		},
	}
	contactRepo := &mockContactRepo{
		getByPhoneFunc: func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
			return &contact.Contact{ID: 10, TenantID: 1, FirstName: "Contact", Whatsapp: &phone}, nil
		},
	}
	profileRepo := &mockProfileRepo{
		getByExternalIDAndChannelIDFunc: func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
			return &profile.Profile{ID: 20, ExternalID: externalID, ChannelID: channelID}, nil
		},
	}
	channelRepo := &mockChannelRepo{
		getByTypeFunc: func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
			return &channel.Channel{ID: 1, Type: "whatsapp"}, nil
		},
	}
	convRepo := &mockConvRepo{
		findOpenByProfileAndChannelFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, profileID int, channelID int) (*conversation.Conversation, error) {
			return &conversation.Conversation{ID: 30, ProfileID: 20, Status: "unassigned"}, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
			return nil
		},
	}
	msgRepo := &mockMsgRepo{
		createFunc: func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) { return 400, nil },
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
	if ev.tenantID != 1 {
		t.Errorf("expected tenantID 1, got %d", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	msg, ok := ev.data.(*message.Message)
	if !ok {
		t.Fatalf("expected data *message.Message, got %T", ev.data)
	}
	if msg.ID != 400 {
		t.Errorf("expected message id 400, got %d", msg.ID)
	}
	if msg.ConversationID != 30 {
		t.Errorf("expected conversation_id 30, got %d", msg.ConversationID)
	}
	if msg.SenderType != "user" {
		t.Errorf("expected sender_type user, got %s", msg.SenderType)
	}
}

func TestHandleMessageUpdate_Delivered(t *testing.T) {
	var capturedID, capturedStatus string
	msgRepo := &mockMsgRepo{
		updateStatusByWebhookIDFunc: func(ctx context.Context, q message.DBTX, webhookMessageID string, status string) error {
			capturedID = webhookMessageID; capturedStatus = status; return nil
		},
	}
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)

	data, _ := json.Marshal(EvolutionMessageUpdate{
		KeyID: "webhook_001", Status: "DELIVERED",
	})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{
		Event: "messages.update", Instance: "instance_test", Data: data,
	})
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if capturedID != "webhook_001" { t.Errorf("expected webhook_001, got %s", capturedID) }
	if capturedStatus != "delivered" { t.Errorf("expected 'delivered', got %s", capturedStatus) }
}

func TestHandleMessageUpdate_Read_Failed_Sent(t *testing.T) {
	tests := []struct{ status, expected string }{
		{"READ", "read"}, {"FAILED", "failed"}, {"SENT", "sent"}, {"DELIVERY_ACK", "delivered"}, {"PLAYED", "read"},
	}
	for _, tt := range tests {
		var capturedStatus string
		msgRepo := &mockMsgRepo{
			updateStatusByWebhookIDFunc: func(ctx context.Context, q message.DBTX, webhookMessageID string, status string) error {
				capturedStatus = status; return nil
			},
		}
		svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)
		data, _ := json.Marshal(EvolutionMessageUpdate{
			KeyID: "w_" + tt.status, Status: tt.status,
		})
		svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "test", Data: data})
		if capturedStatus != tt.expected { t.Errorf("for %s: expected %s, got %s", tt.status, tt.expected, capturedStatus) }
	}
}

func TestHandleMessageUpdate_UnhandledStatus(t *testing.T) {
	called := false
	msgRepo := &mockMsgRepo{
		updateStatusByWebhookIDFunc: func(ctx context.Context, q message.DBTX, webhookMessageID string, status string) error {
			called = true; return nil
		},
	}
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)
	data, _ := json.Marshal(EvolutionMessageUpdate{
		KeyID: "w_004", Status: "PENDING",
	})
	svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update", Instance: "test", Data: data})
	if called { t.Error("expected no call for unhandled status") }
}

func TestHandleMessageUpdate_NilData(t *testing.T) {
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "messages.update"})
	if err != nil { t.Fatalf("expected nil error for nil data, got: %v", err) }
}

func TestHandleConnectionUpdate_Open(t *testing.T) {
	var updatedDevice *whatsapp_device.WhatsAppDevice
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: 1, TenantID: 1, WhatsappID: "instance_test", Status: "DISCONNECTED"}, nil
		},
		updateFunc: func(ctx context.Context, q whatsapp_device.DBTX, tenantID int, id int, device *whatsapp_device.WhatsAppDevice) error {
			updatedDevice = device; return nil
		},
	}
	svc := newTestService(deviceRepo, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})

	data, _ := json.Marshal(EvolutionConnectionUpdate{Instance: "instance_test", State: "open"})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "connection.update", Instance: "instance_test", Data: data})
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if updatedDevice == nil || updatedDevice.Status != "CONNECTED" {
		t.Errorf("expected status CONNECTED, got %v", updatedDevice.Status)
	}
}

func TestHandleConnectionUpdate_Disconnected(t *testing.T) {
	var updatedDevice *whatsapp_device.WhatsAppDevice
	deviceRepo := &mockDeviceRepo{
		getByWhatsappIDFunc: func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
			return &whatsapp_device.WhatsAppDevice{ID: 1, TenantID: 1, WhatsappID: "instance_test", Status: "CONNECTED"}, nil
		},
		updateFunc: func(ctx context.Context, q whatsapp_device.DBTX, tenantID int, id int, device *whatsapp_device.WhatsAppDevice) error {
			updatedDevice = device; return nil
		},
	}
	svc := newTestService(deviceRepo, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})

	data, _ := json.Marshal(EvolutionConnectionUpdate{Instance: "instance_test", State: "close"})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "connection.update", Instance: "instance_test", Data: data})
	if err != nil { t.Fatalf("unexpected error: %v", err) }
	if updatedDevice == nil || updatedDevice.Status != "DISCONNECTED" {
		t.Errorf("expected status DISCONNECTED, got %v", updatedDevice.Status)
	}
}

func TestHandleConnectionUpdate_NilData(t *testing.T) {
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, &mockMsgRepo{})
	err := svc.ProcessEvolutionEvent(context.Background(), &EvolutionWebhookPayload{Event: "connection.update"})
	if err != nil { t.Fatalf("expected nil error for nil data, got: %v", err) }
}
