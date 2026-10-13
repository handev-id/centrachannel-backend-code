package message

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/messenger"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/conversation"
	"centrachannel/internal/src/profile"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

// mock SQL driver for creating *sql.DB without a real database

func init() {
	for _, name := range sql.Drivers() {
		if name == "mock" {
			return
		}
	}
	sql.Register("mock", &mockSQLDriver{})
}

type mockSQLDriver struct{}

func (d *mockSQLDriver) Open(string) (driver.Conn, error) {
	return &mockSQLConn{}, nil
}

type mockSQLConn struct{}

func (c *mockSQLConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *mockSQLConn) Close() error {
	return nil
}

func (c *mockSQLConn) Begin() (driver.Tx, error) {
	return &mockSQLTx{}, nil
}

type mockSQLTx struct{}

func (t *mockSQLTx) Commit() error   { return nil }
func (t *mockSQLTx) Rollback() error { return nil }

// mock repositories

type mockMessageRepository struct {
	getByIDFunc func(context.Context, DBTX, uuid.UUID, uuid.UUID, uuid.UUID) (*Message, error)
	MessageRepository
	createFunc                  func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error)
	listCursorFunc              func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error)
	updateStatusFunc            func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error
	updateStatusByWebhookIDFunc func(ctx context.Context, q DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error
}

func (m *mockMessageRepository) GetByID(ctx context.Context, q DBTX, tenantID, conversationID, id uuid.UUID) (*Message, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, tenantID, conversationID, id)
	}
	return nil, fmt.Errorf("message not found")
}

func (m *mockMessageRepository) Create(ctx context.Context, db *sql.DB, msg *Message, lastMessageJSON []byte, lastAgentID *uuid.UUID, conversations conversation.ConversationRepository) (uuid.UUID, error) {

	id, err := m.createFunc(ctx, db, msg)
	if err != nil {
		return uuid.Nil, err
	}
	if conversations != nil {
		if err := conversations.UpdateLastMessage(ctx, db, msg.TenantID, msg.ConversationID, lastMessageJSON, lastAgentID); err != nil {
			return uuid.Nil, err
		}
	}
	return id, nil

}

func (m *mockMessageRepository) ListCursor(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error) {
	return m.listCursorFunc(ctx, q, tenantID, conversationID, limit, lastID)
}

func (m *mockMessageRepository) UpdateStatus(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	return m.updateStatusFunc(ctx, q, tenantID, id, status)
}

func (m *mockMessageRepository) UpdateStatusByWebhookID(ctx context.Context, q DBTX, tenantID uuid.UUID, webhookMessageID string, status string) error {
	if m.updateStatusByWebhookIDFunc != nil {
		return m.updateStatusByWebhookIDFunc(ctx, q, tenantID, webhookMessageID, status)
	}
	return nil
}

func (m *mockMessageRepository) UpdateWebhookID(ctx context.Context, db *sql.DB, tenantID uuid.UUID, id uuid.UUID, webhookMessageID string) error {
	return nil
}

func (m *mockMessageRepository) MarkReadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) error {
	return nil
}

func (m *mockMessageRepository) GetUnreadByConversation(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID) ([]*Message, error) {
	return nil, nil
}

type mockConversationRepository struct {
	updateLastMessageFunc func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error
}

func (m *mockConversationRepository) GetByID(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) (*conversation.Conversation, error) {
	return &conversation.Conversation{ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), ChannelID: uuid.MustParse("00000000-0000-4000-8000-000000000001")}, nil
}

func (m *mockConversationRepository) Create(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (uuid.UUID, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) UpdateStatus(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) Assign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, agentID uuid.UUID) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) Unassign(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) MarkRead(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) GetTotalUnread(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID) (int, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) ListCursor(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, limit int, status string, channelID, agentID uuid.UUID, search string, lastActivity *time.Time, lastID uuid.UUID) ([]*conversation.Conversation, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) UpdateLastMessage(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
	return m.updateLastMessageFunc(ctx, q, tenantID, id, lastMessageJSON, lastAgentID)
}

func (m *mockConversationRepository) FindOpenByProfileAndChannel(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, profileID uuid.UUID, channelID uuid.UUID) (*conversation.Conversation, error) {
	return nil, nil
}

// mockProfileRepository
type mockProfileRepository struct {
	getByExternalIDAndChannelIDFunc func(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error)
	createFunc                      func(ctx context.Context, q profile.DBTX, profile *profile.Profile) (uuid.UUID, error)
}

func (m *mockProfileRepository) GetByContactIDs(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactIDs []uuid.UUID) (map[uuid.UUID][]profile.Profile, error) {
	return map[uuid.UUID][]profile.Profile{}, nil
}
func (m *mockProfileRepository) List(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID, channelID uuid.UUID) ([]profile.Profile, error) {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID) (*profile.Profile, error) {
	return &profile.Profile{ExternalID: "test_external_id"}, nil
}
func (m *mockProfileRepository) Update(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, id uuid.UUID, p *profile.Profile) error {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByContactID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, contactID uuid.UUID) ([]profile.Profile, error) {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByExternalIDAndChannelID(ctx context.Context, q profile.DBTX, tenantID uuid.UUID, externalID string, channelID uuid.UUID) (*profile.Profile, error) {
	if m.getByExternalIDAndChannelIDFunc != nil {
		return m.getByExternalIDAndChannelIDFunc(ctx, q, tenantID, externalID, channelID)
	}
	return nil, nil
}
func (m *mockProfileRepository) Create(ctx context.Context, q profile.DBTX, p *profile.Profile) (uuid.UUID, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, p)
	}
	return uuid.Nil, nil
}

// mockChannelRepository
type mockChannelRepository struct {
	getByIDFunc   func(context.Context, channel.DBTX, uuid.UUID) (*channel.Channel, error)
	getByTypeFunc func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error)
}

func (m *mockChannelRepository) List(ctx context.Context, q channel.DBTX) ([]channel.Channel, error) {
	panic("unexpected call")
}
func (m *mockChannelRepository) GetByID(ctx context.Context, q channel.DBTX, id uuid.UUID) (*channel.Channel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, id)
	}
	return &channel.Channel{Type: "facebook"}, nil
}
func (m *mockChannelRepository) GetByType(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
	if m.getByTypeFunc != nil {
		return m.getByTypeFunc(ctx, q, channelType)
	}
	return nil, nil
}

// mockTenantRepository
type mockTenantRepository struct{}

func (m *mockTenantRepository) Create(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) (uuid.UUID, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) CreateRole(ctx context.Context, q tenant.DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) CreateUser(ctx context.Context, q tenant.DBTX, user *tenant.User) (uuid.UUID, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) AttachRole(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID uuid.UUID) error {
	panic("unexpected call")
}
func (m *mockTenantRepository) List(ctx context.Context, q tenant.DBTX) ([]tenant.Tenant, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) GetByID(ctx context.Context, q tenant.DBTX, id uuid.UUID) (*tenant.Tenant, error) {
	return &tenant.Tenant{}, nil
}
func (m *mockTenantRepository) GetByMetaPageID(ctx context.Context, q tenant.DBTX, pageID string) (*tenant.Tenant, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) GetByMetaInstagramBusinessID(ctx context.Context, q tenant.DBTX, igID string) (*tenant.Tenant, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) Update(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) error {
	panic("unexpected call")
}

// mockMessenger
type mockMessenger struct{}

func (m *mockMessenger) Send(msg *messenger.OutgoingMessage) (string, error) {
	return "ext_msg_123", nil
}

// helpers

func strPtr(s string) *string { return &s }

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

// tests

func TestMessageService_Send(t *testing.T) {
	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	t.Run("sender type user passes lastAgentID to UpdateLastMessage", func(t *testing.T) {
		var capturedCreateMsg *Message
		var capturedLastAgentID *uuid.UUID

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				capturedCreateMsg = msg
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastAgentID = lastAgentID
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("hello"),
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-00000000002a"),
			SenderType: "user",
		}
		msg, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected ID 1, got %s", msg.ID)
		}
		if capturedCreateMsg == nil {
			t.Fatal("expected Create to be called")
		}
		if capturedCreateMsg.TenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected TenantID 1, got %s", capturedCreateMsg.TenantID)
		}
		if capturedCreateMsg.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
			t.Errorf("expected ConversationID 10, got %s", capturedCreateMsg.ConversationID)
		}
		if capturedCreateMsg.SenderType != "user" {
			t.Errorf("expected SenderType user, got %s", capturedCreateMsg.SenderType)
		}
		if capturedCreateMsg.SenderID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
			t.Errorf("expected SenderID 42, got %s", capturedCreateMsg.SenderID)
		}
		if capturedCreateMsg.Status != "pending" {
			t.Errorf("expected Status pending, got %s", capturedCreateMsg.Status)
		}
		if capturedLastAgentID == nil || *capturedLastAgentID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
			t.Errorf("expected lastAgentID 42 for user sender, got %v", capturedLastAgentID)
		}
	})

	t.Run("sender type contact passes nil as lastAgentID", func(t *testing.T) {
		var capturedLastAgentID *uuid.UUID

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastAgentID = lastAgentID
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("hello"),
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-000000000063"),
			SenderType: "contact",
		}
		_, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLastAgentID != nil {
			t.Errorf("expected nil lastAgentID for contact sender, got %s", *capturedLastAgentID)
		}
	})

	t.Run("attachment is included in last_message JSON when present", func(t *testing.T) {
		var capturedLastMsgJSON []byte

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastMsgJSON = lastMessageJSON
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("with attachment"),
			Attachment: json.RawMessage(`{"url":"http://example.com/file.pdf","type":"document","mimetype":"application/pdf","size":100}`),
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			SenderType: "user",
		}
		_, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(capturedLastMsgJSON, &result); err != nil {
			t.Fatalf("failed to unmarshal lastMsgJSON: %v", err)
		}
		att, ok := result["attachment"]
		if !ok {
			t.Fatal("expected attachment key in last_message JSON")
		}
		attMap, ok := att.(map[string]interface{})
		if !ok {
			t.Fatal("expected attachment to be an object")
		}
		if attMap["url"] != "http://example.com/file.pdf" {
			t.Errorf("expected url http://example.com/file.pdf, got %v", attMap["url"])
		}
		if attMap["type"] != "document" {
			t.Errorf("expected type document, got %v", attMap["type"])
		}
	})

	t.Run("attachment omitted from last_message JSON when absent", func(t *testing.T) {
		var capturedLastMsgJSON []byte

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
				return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				capturedLastMsgJSON = lastMessageJSON
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, &mockNotifier{})
		req := SendMessageRequest{
			Text:       strPtr("no attachment"),
			Attachment: nil,
			SenderID:   uuid.MustParse("00000000-0000-4000-8000-000000000001"),
			SenderType: "user",
		}
		_, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(capturedLastMsgJSON, &result); err != nil {
			t.Fatalf("failed to unmarshal lastMsgJSON: %v", err)
		}
		if _, ok := result["attachment"]; ok {
			t.Error("attachment should not appear in last_message when absent from request")
		}
	})
}

func TestMessageService_Send_NotifiesNewMessage(t *testing.T) {
	notifier := &mockNotifier{}

	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	msgRepo := &mockMessageRepository{
		createFunc: func(ctx context.Context, q DBTX, msg *Message) (uuid.UUID, error) {
			return uuid.MustParse("00000000-0000-4000-8000-000000000001"), nil
		},
	}
	convRepo := &mockConversationRepository{
		updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
			return nil
		},
	}

	svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil, notifier)
	req := SendMessageRequest{
		Text:       strPtr("hello"),
		SenderID:   uuid.MustParse("00000000-0000-4000-8000-00000000002a"),
		SenderType: "user",
	}
	msg, err := svc.Send(ctx, req, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-00000000000a"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected ID 1, got %s", msg.ID)
	}

	ev := notifier.last()
	if ev == nil {
		t.Fatal("expected a websocket notification")
	}
	if ev.tenantID != uuid.MustParse("00000000-0000-4000-8000-000000000007") {
		t.Errorf("expected tenantID 7, got %s", ev.tenantID)
	}
	if ev.event != "new-message" {
		t.Errorf("expected event 'new-message', got %q", ev.event)
	}
	notified, ok := ev.data.(*Message)
	if !ok {
		t.Fatalf("expected data *Message, got %T", ev.data)
	}
	if notified.ID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected message id 1, got %s", notified.ID)
	}
	if notified.ConversationID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
		t.Errorf("expected conversation_id 10, got %s", notified.ConversationID)
	}
	if notified.SenderType != "user" {
		t.Errorf("expected sender_type user, got %s", notified.SenderType)
	}
}

func TestMessageService_ListCursor(t *testing.T) {
	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	t.Run("defaults_limit_and_passes_cursor", func(t *testing.T) {
		var capturedLimit int
		var capturedLastID uuid.UUID
		var capturedTenantID uuid.UUID

		msgRepo := &mockMessageRepository{
			listCursorFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error) {
				capturedTenantID = tenantID
				capturedLimit = limit
				capturedLastID = lastID
				return []*Message{{ID: uuid.MustParse("00000000-0000-4000-8000-000000000032")}, {ID: uuid.MustParse("00000000-0000-4000-8000-000000000031")}, {ID: uuid.MustParse("00000000-0000-4000-8000-000000000030")}}, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		result, lastID, hasMore, err := svc.ListCursor(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), ListMessageQuery{Limit: 0, LastID: uuid.MustParse("00000000-0000-4000-8000-000000000037")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000007") {
			t.Errorf("expected tenantID 7, got %s", capturedTenantID)
		}
		if capturedLimit != 51 {
			t.Errorf("expected limit+1 = 51, got %d", capturedLimit)
		}
		if capturedLastID != uuid.MustParse("00000000-0000-4000-8000-000000000037") {
			t.Errorf("expected lastID 55, got %s", capturedLastID)
		}
		if hasMore {
			t.Error("expected hasMore false")
		}
		// repo returns DESC (newest first); service reverses to ascending chronological order
		if len(result) != 3 {
			t.Fatalf("expected 3 messages, got %d", len(result))
		}
		if result[0].ID != uuid.MustParse("00000000-0000-4000-8000-000000000030") || result[1].ID != uuid.MustParse("00000000-0000-4000-8000-000000000031") || result[2].ID != uuid.MustParse("00000000-0000-4000-8000-000000000032") {
			t.Errorf("expected reversed ascending order [48,49,50], got [%s,%s,%s]", result[0].ID, result[1].ID, result[2].ID)
		}
		if lastID != uuid.MustParse("00000000-0000-4000-8000-000000000030") {
			t.Errorf("expected cursor lastID 48 (oldest kept), got %s", lastID)
		}
	})

	t.Run("has_more_trims_and_keeps_newest", func(t *testing.T) {
		var capturedLimit int
		msgs := make([]*Message, 0, 51)
		for i := 51; i >= 1; i-- {
			msgs = append(msgs, &Message{ID: uuid.UUID{6: 0x40, 8: 0x80, 15: byte(i)}})
		}

		msgRepo := &mockMessageRepository{
			listCursorFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, conversationID uuid.UUID, limit int, lastID uuid.UUID) ([]*Message, error) {
				capturedLimit = limit
				return msgs, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		result, lastID, hasMore, err := svc.ListCursor(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000007"), uuid.MustParse("00000000-0000-4000-8000-000000000001"), ListMessageQuery{Limit: 50, LastID: uuid.MustParse("00000000-0000-4000-8000-00000000003c")})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLimit != 51 {
			t.Errorf("expected limit+1 = 51, got %d", capturedLimit)
		}
		if !hasMore {
			t.Error("expected hasMore true")
		}
		if len(result) != 50 {
			t.Fatalf("expected 50 messages, got %d", len(result))
		}
		if result[0].ID != uuid.MustParse("00000000-0000-4000-8000-000000000002") || result[49].ID != uuid.MustParse("00000000-0000-4000-8000-000000000033") {
			t.Errorf("expected ascending [2..51], got first=%s last=%s", result[0].ID, result[49].ID)
		}
		if lastID != uuid.MustParse("00000000-0000-4000-8000-000000000002") {
			t.Errorf("expected cursor lastID 2 (oldest kept), got %s", lastID)
		}
	})
}

func TestMessageService_UpdateStatus(t *testing.T) {
	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	t.Run("delegates to repository", func(t *testing.T) {
		var capturedID uuid.UUID
		var capturedStatus string

		msgRepo := &mockMessageRepository{
			updateStatusFunc: func(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
				capturedID = id
				capturedStatus = status
				return nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID uuid.UUID, id uuid.UUID, lastMessageJSON []byte, lastAgentID *uuid.UUID) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		if err := svc.UpdateStatus(ctx, uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-000000000005"), "read"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedID != uuid.MustParse("00000000-0000-4000-8000-000000000005") {
			t.Errorf("expected id 5, got %s", capturedID)
		}
		if capturedStatus != "read" {
			t.Errorf("expected status 'read', got %s", capturedStatus)
		}
	})
}

func (m *mockTenantRepository) GetDomains(ctx context.Context, q tenant.DBTX, id uuid.UUID) (json.RawMessage, error) {
	return json.RawMessage("[]"), nil
}
func (m *mockTenantRepository) GetActiveIDByDomain(ctx context.Context, q tenant.DBTX, name string) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockMessageRepository) RetryFailed(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, senderID uuid.UUID) (*Message, error) {
	return nil, fmt.Errorf("not found")
}

func TestSendReplyUsesScopedTarget(t *testing.T) {
	tenantID, conversationID, targetID := uuid.New(), uuid.New(), uuid.New()
	providerID, original := "original-provider", "Original"
	repo := &mockMessageRepository{createFunc: func(_ context.Context, _ DBTX, msg *Message) (uuid.UUID, error) { return uuid.New(), nil }, getByIDFunc: func(_ context.Context, _ DBTX, tenant, conv, id uuid.UUID) (*Message, error) {
		if tenant != tenantID || conv != conversationID || id != targetID {
			t.Fatal("reply target lookup not scoped")
		}
		return &Message{ID: targetID, TenantID: tenantID, ConversationID: conversationID, WebhookMessageID: &providerID, Text: &original, SenderType: "contact"}, nil
	}}
	channels := &mockChannelRepository{getByIDFunc: func(context.Context, channel.DBTX, uuid.UUID) (*channel.Channel, error) {
		return &channel.Channel{Type: "whatsapp"}, nil
	}}
	repo.updateStatusFunc = func(context.Context, DBTX, uuid.UUID, uuid.UUID, string) error { return nil }
	convs := &mockConversationRepository{updateLastMessageFunc: func(context.Context, conversation.DBTX, uuid.UUID, uuid.UUID, []byte, *uuid.UUID) error { return nil }}
	svc := NewMessageService(repo, convs, &mockProfileRepository{}, channels, &mockTenantRepository{}, nil, &config.Config{}, logger.NewLogger("error", "text"), nil)
	req := SendMessageRequest{Text: strPtr("Reply"), SenderID: uuid.New(), SenderType: "user"}
	if err := json.Unmarshal([]byte(`{"reply_to_id":"`+targetID.String()+`"}`), &req); err != nil {
		t.Fatal(err)
	}
	sent, err := svc.Send(context.Background(), req, tenantID, conversationID)
	if err != nil {
		t.Fatal(err)
	}
	if sent.Reply == nil || sent.Reply.ProviderMessageID != providerID || sent.Reply.MessageID == nil || *sent.Reply.MessageID != targetID {
		t.Fatalf("sent reply lacks original context: %+v", sent)
	}
}
