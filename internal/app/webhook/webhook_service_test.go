package webhook

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/contact"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/message"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/src/whatsapp_device"
	"centrachannel/internal/utils/logger"
)

// ---- mock repositories ----

type mockDeviceRepo struct {
	getByWhatsappIDFunc func(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error)
	updateFunc          func(ctx context.Context, q whatsapp_device.DBTX, tenantID int, id int, device *whatsapp_device.WhatsAppDevice) error
}

func (m *mockDeviceRepo) List(ctx context.Context, q whatsapp_device.DBTX, tenantID int) ([]whatsapp_device.WhatsAppDevice, error) { return nil, nil }
func (m *mockDeviceRepo) GetByID(ctx context.Context, q whatsapp_device.DBTX, tenantID int, id int) (*whatsapp_device.WhatsAppDevice, error) { return nil, nil }
func (m *mockDeviceRepo) GetByWhatsappID(ctx context.Context, q whatsapp_device.DBTX, whatsappID string) (*whatsapp_device.WhatsAppDevice, error) {
	if m.getByWhatsappIDFunc != nil { return m.getByWhatsappIDFunc(ctx, q, whatsappID) }
	return nil, nil
}
func (m *mockDeviceRepo) Create(ctx context.Context, q whatsapp_device.DBTX, device *whatsapp_device.WhatsAppDevice) (int, error) { return 0, nil }
func (m *mockDeviceRepo) Update(ctx context.Context, q whatsapp_device.DBTX, tenantID int, id int, device *whatsapp_device.WhatsAppDevice) error {
	if m.updateFunc != nil { return m.updateFunc(ctx, q, tenantID, id, device) }
	return nil
}
func (m *mockDeviceRepo) Delete(ctx context.Context, q whatsapp_device.DBTX, tenantID int, id int) error { return nil }

type mockContactRepo struct {
	getByPhoneFunc func(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error)
	createFunc     func(ctx context.Context, q contact.DBTX, c *contact.Contact) (int, error)
}

func (m *mockContactRepo) List(ctx context.Context, q contact.DBTX, tenantID int, limit, offset int, search, status string, channelID int) ([]*contact.Contact, int, error) { return nil, 0, nil }
func (m *mockContactRepo) GetByID(ctx context.Context, q contact.DBTX, tenantID int, id int) (*contact.Contact, error) { return nil, nil }
func (m *mockContactRepo) GetByPhone(ctx context.Context, q contact.DBTX, tenantID int, phone string) (*contact.Contact, error) {
	if m.getByPhoneFunc != nil { return m.getByPhoneFunc(ctx, q, tenantID, phone) }
	return nil, nil
}
func (m *mockContactRepo) Create(ctx context.Context, q contact.DBTX, c *contact.Contact) (int, error) {
	if m.createFunc != nil { return m.createFunc(ctx, q, c) }
	return 0, nil
}
func (m *mockContactRepo) Update(ctx context.Context, q contact.DBTX, tenantID int, id int, c *contact.Contact) error { return nil }
func (m *mockContactRepo) SoftDelete(ctx context.Context, q contact.DBTX, tenantID int, id int) error { return nil }

type mockProfileRepo struct {
	getByExternalIDAndChannelIDFunc func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error)
	createFunc                      func(ctx context.Context, q profile.DBTX, p *profile.Profile) (int, error)
}

func (m *mockProfileRepo) List(ctx context.Context, q profile.DBTX, contactID, channelID int) ([]profile.Profile, error) { return nil, nil }
func (m *mockProfileRepo) GetByID(ctx context.Context, q profile.DBTX, id int) (*profile.Profile, error) { return nil, nil }
func (m *mockProfileRepo) GetByExternalIDAndChannelID(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
	if m.getByExternalIDAndChannelIDFunc != nil { return m.getByExternalIDAndChannelIDFunc(ctx, q, externalID, channelID) }
	return nil, nil
}
func (m *mockProfileRepo) Create(ctx context.Context, q profile.DBTX, p *profile.Profile) (int, error) {
	if m.createFunc != nil { return m.createFunc(ctx, q, p) }
	return 0, nil
}
func (m *mockProfileRepo) Update(ctx context.Context, q profile.DBTX, id int, p *profile.Profile) error { return nil }
func (m *mockProfileRepo) GetByContactID(ctx context.Context, q profile.DBTX, contactID int) ([]profile.Profile, error) { return nil, nil }

type mockChannelRepo struct {
	getByTypeFunc func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error)
}

func (m *mockChannelRepo) GetByID(ctx context.Context, q channel.DBTX, id int) (*channel.Channel, error) { return &channel.Channel{}, nil }
func (m *mockChannelRepo) GetByType(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
	if m.getByTypeFunc != nil { return m.getByTypeFunc(ctx, q, channelType) }
	return &channel.Channel{}, nil
}
func (m *mockChannelRepo) List(ctx context.Context, q channel.DBTX) ([]channel.Channel, error) { return nil, nil }

type mockConvRepo struct {
	listFunc             func(ctx context.Context, q conversation.DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*conversation.Conversation, int, error)
	createFunc           func(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (int, error)
	updateLastMessageFunc func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID int) error
}

func (m *mockConvRepo) List(ctx context.Context, q conversation.DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*conversation.Conversation, int, error) {
	if m.listFunc != nil { return m.listFunc(ctx, q, tenantID, limit, offset, status, channelID, agentID, search) }
	return nil, 0, nil
}
func (m *mockConvRepo) GetByID(ctx context.Context, q conversation.DBTX, tenantID int, id int) (*conversation.Conversation, error) { return nil, nil }
func (m *mockConvRepo) Create(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (int, error) {
	if m.createFunc != nil { return m.createFunc(ctx, q, conv) }
	return 0, nil
}
func (m *mockConvRepo) UpdateStatus(ctx context.Context, q conversation.DBTX, tenantID int, id int, status string) error { return nil }
func (m *mockConvRepo) Assign(ctx context.Context, q conversation.DBTX, tenantID int, id int, agentID int) error { return nil }
func (m *mockConvRepo) Unassign(ctx context.Context, q conversation.DBTX, tenantID int, id int) error { return nil }
func (m *mockConvRepo) GetTotalUnread(ctx context.Context, q conversation.DBTX, tenantID int) (int, error) { return 0, nil }
func (m *mockConvRepo) MarkRead(ctx context.Context, q conversation.DBTX, tenantID int, id int) error { return nil }
func (m *mockConvRepo) ListCursor(ctx context.Context, q conversation.DBTX, tenantID int, limit int, status string, channelID, agentID int, search string, lastActivity *time.Time, lastID int) ([]*conversation.Conversation, error) {
	return nil, nil
}
func (m *mockConvRepo) UpdateLastMessage(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID int) error {
	if m.updateLastMessageFunc != nil { return m.updateLastMessageFunc(ctx, q, tenantID, id, lastMessageJSON, lastAgentID) }
	return nil
}

type mockMsgRepo struct {
	createFunc                func(ctx context.Context, q message.DBTX, msg *message.Message) (int, error)
	updateStatusByWebhookIDFunc func(ctx context.Context, q message.DBTX, webhookMessageID string, status string) error
}

func (m *mockMsgRepo) List(ctx context.Context, q message.DBTX, conversationID int, limit, offset int) ([]*message.Message, int, error) { return nil, 0, nil }
func (m *mockMsgRepo) Create(ctx context.Context, q message.DBTX, msg *message.Message) (int, error) {
	if m.createFunc != nil { return m.createFunc(ctx, q, msg) }
	return 0, nil
}
func (m *mockMsgRepo) UpdateStatus(ctx context.Context, q message.DBTX, id int, status string) error { return nil }
func (m *mockMsgRepo) UpdateStatusByWebhookID(ctx context.Context, q message.DBTX, webhookMessageID string, status string) error {
	if m.updateStatusByWebhookIDFunc != nil { return m.updateStatusByWebhookIDFunc(ctx, q, webhookMessageID, status) }
	return nil
}
func (m *mockMsgRepo) ListCursor(ctx context.Context, q message.DBTX, conversationID int, limit int, lastID int) ([]*message.Message, error) { return nil, nil }
func (m *mockMsgRepo) UpdateWebhookID(ctx context.Context, q message.DBTX, id int, webhookMessageID string) error { return nil }

// ---- test helpers ----

type mockMetaTenantRepo struct{}

func (m *mockMetaTenantRepo) Create(ctx context.Context, q tenant.DBTX, t *tenant.Tenant) (int, error) { return 0, nil }
func (m *mockMetaTenantRepo) CreateRole(ctx context.Context, q tenant.DBTX, tenantID int, name string) (int, error) { return 0, nil }
func (m *mockMetaTenantRepo) CreateUser(ctx context.Context, q tenant.DBTX, u *tenant.User) (int, error) { return 0, nil }
func (m *mockMetaTenantRepo) AttachRole(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID int) error { return nil }
func (m *mockMetaTenantRepo) List(ctx context.Context, q tenant.DBTX) ([]tenant.Tenant, error) { return nil, nil }
func (m *mockMetaTenantRepo) GetByID(ctx context.Context, q tenant.DBTX, id int) (*tenant.Tenant, error) { return nil, nil }
func (m *mockMetaTenantRepo) GetByMetaPageID(ctx context.Context, q tenant.DBTX, pageID string) (*tenant.Tenant, error) {
	return &tenant.Tenant{ID: 1, Name: "Test Tenant"}, nil
}
func (m *mockMetaTenantRepo) GetByMetaInstagramBusinessID(ctx context.Context, q tenant.DBTX, igID string) (*tenant.Tenant, error) {
	return &tenant.Tenant{ID: 1, Name: "Test Tenant"}, nil
}

func newTestService(deviceRepo whatsapp_device.WhatsAppDeviceRepository, contactRepo contact.ContactRepository, profileRepo profile.ProfileRepository, channelRepo channel.ChannelRepository, convRepo conversation.ConversationRepository, msgRepo message.MessageRepository) WebhookService {
	return NewWebhookService(deviceRepo, contactRepo, profileRepo, channelRepo, convRepo, msgRepo, &mockMetaTenantRepo{}, nil, logger.NewLogger("debug", "text"), nil)
}

// ---- tests ----

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
		listFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*conversation.Conversation, int, error) {
			return []*conversation.Conversation{{ID: 30, ProfileID: 20, Status: "unassigned"}}, 1, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID int) error { return nil },
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
		listFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*conversation.Conversation, int, error) {
			return nil, 0, nil
		},
		createFunc: func(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (int, error) {
			createdConv = conv
			return 300, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID int) error {
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
		listFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*conversation.Conversation, int, error) {
			return []*conversation.Conversation{{ID: 30, ProfileID: 20, Status: "unassigned"}}, 1, nil
		},
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID int) error { return nil },
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

func TestHandleMessageUpdate_Delivered(t *testing.T) {
	var capturedID, capturedStatus string
	msgRepo := &mockMsgRepo{
		updateStatusByWebhookIDFunc: func(ctx context.Context, q message.DBTX, webhookMessageID string, status string) error {
			capturedID = webhookMessageID; capturedStatus = status; return nil
		},
	}
	svc := newTestService(&mockDeviceRepo{}, &mockContactRepo{}, &mockProfileRepo{}, &mockChannelRepo{}, &mockConvRepo{}, msgRepo)

	data, _ := json.Marshal(EvolutionMessageUpdate{
		Key: EvolutionMessageKey{ID: "webhook_001"}, Update: EvolutionStatusUpdate{Status: "DELIVERED"},
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
		{"READ", "read"}, {"FAILED", "failed"}, {"SENT", "sent"},
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
			Key: EvolutionMessageKey{ID: "w_" + tt.status}, Update: EvolutionStatusUpdate{Status: tt.status},
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
		Key: EvolutionMessageKey{ID: "w_004"}, Update: EvolutionStatusUpdate{Status: "PENDING"},
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

func TestExtractMessageContent(t *testing.T) {
	svc := &webhookService{}

	// conversation
	text := "Hello"
	txt, att := svc.extractMessageContent(EvolutionMessage{Conversation: &text}, "conversation")
	if txt == nil || *txt != "Hello" { t.Errorf("expected 'Hello', got %v", txt) }
	if att != nil { t.Error("expected nil attachment") }

	// extendedTextMessage
	msg := EvolutionMessage{ExtendedTextMessage: &struct{ Text string `json:"text"` }{Text: "Extended"}}
	txt, att = svc.extractMessageContent(msg, "extendedTextMessage")
	if txt == nil || *txt != "Extended" { t.Errorf("expected 'Extended', got %v", txt) }
	if att != nil { t.Error("expected nil attachment") }

	// image with caption
	imgMsg := EvolutionMessage{ImageMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Caption string `json:"caption,omitempty"`
	}{URL: "https://img.url/p.jpg", Mimetype: "image/jpeg", Caption: "Nice pic"}}
	txt, att = svc.extractMessageContent(imgMsg, "imageMessage")
	if txt == nil || *txt != "Nice pic" { t.Errorf("expected 'Nice pic', got %v", txt) }
	if att == nil { t.Fatal("expected attachment") }

	// image without caption
	imgNoCap := EvolutionMessage{ImageMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Caption string `json:"caption,omitempty"`
	}{URL: "https://img.url/p.jpg", Mimetype: "image/jpeg"}}
	txt, att = svc.extractMessageContent(imgNoCap, "imageMessage")
	if txt != nil { t.Errorf("expected nil text, got %v", *txt) }
	if att == nil { t.Fatal("expected attachment") }

	// video
	vidMsg := EvolutionMessage{VideoMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; Caption string `json:"caption,omitempty"`
	}{URL: "https://vid.url/v.mp4", Mimetype: "video/mp4", Caption: "Check this"}}
	txt, att = svc.extractMessageContent(vidMsg, "videoMessage")
	if txt == nil || *txt != "Check this" { t.Errorf("expected 'Check this', got %v", txt) }
	if att == nil { t.Fatal("expected attachment") }

	// audio
	audMsg := EvolutionMessage{AudioMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`
	}{URL: "https://audio.url/v.ogg", Mimetype: "audio/ogg"}}
	txt, att = svc.extractMessageContent(audMsg, "audioMessage")
	if txt != nil { t.Errorf("expected nil text, got %v", *txt) }
	if att == nil { t.Fatal("expected attachment") }

	// document
	docMsg := EvolutionMessage{DocumentMessage: &struct {
		URL string `json:"url"`; Mimetype string `json:"mimetype"`; FileName string `json:"fileName,omitempty"`
	}{URL: "https://doc.url/r.pdf", Mimetype: "application/pdf", FileName: "report.pdf"}}
	txt, att = svc.extractMessageContent(docMsg, "documentMessage")
	if txt != nil { t.Errorf("expected nil text, got %v", *txt) }
	if att == nil { t.Fatal("expected attachment") }

	// empty
	txt, att = svc.extractMessageContent(EvolutionMessage{}, "")
	if txt != nil || att != nil { t.Error("expected nil text and nil attachment") }
}

func TestExtractPhoneFromJID(t *testing.T) {
	tests := []struct{ input, expected string }{
		{"5511999999999@s.whatsapp.net", "5511999999999"},
		{"5511999999999", "5511999999999"},
		{"", ""},
	}
	for _, tt := range tests {
		result := extractPhoneFromJID(tt.input)
		if result != tt.expected { t.Errorf("extractPhoneFromJID(%q) = %q, want %q", tt.input, result, tt.expected) }
	}
}
