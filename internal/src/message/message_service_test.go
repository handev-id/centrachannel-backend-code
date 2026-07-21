package message

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"
	"time"

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
	createFunc                 func(ctx context.Context, q DBTX, msg *Message) (int, error)
	listFunc                   func(ctx context.Context, q DBTX, conversationID int, limit, offset int) ([]*Message, int, error)
	listCursorFunc             func(ctx context.Context, q DBTX, conversationID int, limit int, lastID int) ([]*Message, error)
	updateStatusFunc           func(ctx context.Context, q DBTX, id int, status string) error
	updateStatusByWebhookIDFunc func(ctx context.Context, q DBTX, webhookMessageID string, status string) error
}

func (m *mockMessageRepository) Create(ctx context.Context, q DBTX, msg *Message) (int, error) {
	return m.createFunc(ctx, q, msg)
}

func (m *mockMessageRepository) List(ctx context.Context, q DBTX, conversationID int, limit, offset int) ([]*Message, int, error) {
	return m.listFunc(ctx, q, conversationID, limit, offset)
}
func (m *mockMessageRepository) ListCursor(ctx context.Context, q DBTX, conversationID int, limit int, lastID int) ([]*Message, error) {
	return m.listCursorFunc(ctx, q, conversationID, limit, lastID)
}

func (m *mockMessageRepository) UpdateStatus(ctx context.Context, q DBTX, id int, status string) error {
	return m.updateStatusFunc(ctx, q, id, status)
}

func (m *mockMessageRepository) UpdateStatusByWebhookID(ctx context.Context, q DBTX, webhookMessageID string, status string) error {
	if m.updateStatusByWebhookIDFunc != nil {
		return m.updateStatusByWebhookIDFunc(ctx, q, webhookMessageID, status)
	}
	return nil
}

func (m *mockMessageRepository) UpdateWebhookID(ctx context.Context, q DBTX, id int, webhookMessageID string) error {
	return nil
}

type mockConversationRepository struct {
	updateLastMessageFunc func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error
}

func (m *mockConversationRepository) List(ctx context.Context, q conversation.DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*conversation.Conversation, int, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) GetByID(ctx context.Context, q conversation.DBTX, tenantID int, id int) (*conversation.Conversation, error) {
	return &conversation.Conversation{ProfileID: 1, ChannelID: 1}, nil
}

func (m *mockConversationRepository) Create(ctx context.Context, q conversation.DBTX, conv *conversation.Conversation) (int, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) UpdateStatus(ctx context.Context, q conversation.DBTX, tenantID int, id int, status string) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) Assign(ctx context.Context, q conversation.DBTX, tenantID int, id int, agentID int) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) Unassign(ctx context.Context, q conversation.DBTX, tenantID int, id int) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) MarkRead(ctx context.Context, q conversation.DBTX, tenantID int, id int) error {
	panic("unexpected call")
}

func (m *mockConversationRepository) GetTotalUnread(ctx context.Context, q conversation.DBTX, tenantID int) (int, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) ListCursor(ctx context.Context, q conversation.DBTX, tenantID int, limit int, status string, channelID, agentID int, search string, lastActivity *time.Time, lastID int) ([]*conversation.Conversation, error) {
	panic("unexpected call")
}

func (m *mockConversationRepository) UpdateLastMessage(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
	return m.updateLastMessageFunc(ctx, q, tenantID, id, lastMessageJSON, lastAgentID)
}

func (m *mockConversationRepository) FindOpenByProfileAndChannel(ctx context.Context, q conversation.DBTX, tenantID int, profileID int, channelID int) (*conversation.Conversation, error) {
	return nil, nil
}

// mockProfileRepository
type mockProfileRepository struct {
	getByExternalIDAndChannelIDFunc func(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error)
	createFunc                      func(ctx context.Context, q profile.DBTX, profile *profile.Profile) (int, error)
}

func (m *mockProfileRepository) GetByContactIDs(ctx context.Context, q profile.DBTX, contactIDs []int) (map[int][]profile.Profile, error) {
	return map[int][]profile.Profile{}, nil
}
func (m *mockProfileRepository) List(ctx context.Context, q profile.DBTX, contactID, channelID int) ([]profile.Profile, error) {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByID(ctx context.Context, q profile.DBTX, id int) (*profile.Profile, error) {
	return &profile.Profile{ExternalID: "test_external_id"}, nil
}
func (m *mockProfileRepository) Update(ctx context.Context, q profile.DBTX, id int, p *profile.Profile) error {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByContactID(ctx context.Context, q profile.DBTX, contactID int) ([]profile.Profile, error) {
	panic("unexpected call")
}
func (m *mockProfileRepository) GetByExternalIDAndChannelID(ctx context.Context, q profile.DBTX, externalID string, channelID int) (*profile.Profile, error) {
	if m.getByExternalIDAndChannelIDFunc != nil {
		return m.getByExternalIDAndChannelIDFunc(ctx, q, externalID, channelID)
	}
	return nil, nil
}
func (m *mockProfileRepository) Create(ctx context.Context, q profile.DBTX, p *profile.Profile) (int, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, p)
	}
	return 0, nil
}

// mockChannelRepository
type mockChannelRepository struct {
	getByTypeFunc func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error)
}

func (m *mockChannelRepository) List(ctx context.Context, q channel.DBTX) ([]channel.Channel, error) {
	panic("unexpected call")
}
func (m *mockChannelRepository) GetByID(ctx context.Context, q channel.DBTX, id int) (*channel.Channel, error) {
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

func (m *mockTenantRepository) Create(ctx context.Context, q tenant.DBTX, tenant *tenant.Tenant) (int, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) CreateRole(ctx context.Context, q tenant.DBTX, tenantID int, name string) (int, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) CreateUser(ctx context.Context, q tenant.DBTX, user *tenant.User) (int, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) AttachRole(ctx context.Context, q tenant.DBTX, tenantID, userID, roleID int) error {
	panic("unexpected call")
}
func (m *mockTenantRepository) List(ctx context.Context, q tenant.DBTX) ([]tenant.Tenant, error) {
	panic("unexpected call")
}
func (m *mockTenantRepository) GetByID(ctx context.Context, q tenant.DBTX, id int) (*tenant.Tenant, error) {
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
		var capturedLastAgentID *int

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (int, error) {
				capturedCreateMsg = msg
				return 1, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
				capturedLastAgentID = lastAgentID
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		req := SendMessageRequest{
			Text:       strPtr("hello"),
			SenderID:   42,
			SenderType: "user",
		}
		msg, err := svc.Send(ctx, req, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if msg.ID != 1 {
			t.Errorf("expected ID 1, got %d", msg.ID)
		}
		if capturedCreateMsg == nil {
			t.Fatal("expected Create to be called")
		}
		if capturedCreateMsg.TenantID != 1 {
			t.Errorf("expected TenantID 1, got %d", capturedCreateMsg.TenantID)
		}
		if capturedCreateMsg.ConversationID != 10 {
			t.Errorf("expected ConversationID 10, got %d", capturedCreateMsg.ConversationID)
		}
		if capturedCreateMsg.SenderType != "user" {
			t.Errorf("expected SenderType user, got %s", capturedCreateMsg.SenderType)
		}
		if capturedCreateMsg.SenderID != 42 {
			t.Errorf("expected SenderID 42, got %d", capturedCreateMsg.SenderID)
		}
		if capturedCreateMsg.Status != "sent" {
			t.Errorf("expected Status sent, got %s", capturedCreateMsg.Status)
		}
		if capturedLastAgentID == nil || *capturedLastAgentID != 42 {
			t.Errorf("expected lastAgentID 42 for user sender, got %v", capturedLastAgentID)
		}
	})

	t.Run("sender type contact passes nil as lastAgentID", func(t *testing.T) {
		var capturedLastAgentID *int

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (int, error) {
				return 1, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
				capturedLastAgentID = lastAgentID
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		req := SendMessageRequest{
			Text:       strPtr("hello"),
			SenderID:   99,
			SenderType: "contact",
		}
		_, err := svc.Send(ctx, req, 1, 10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLastAgentID != nil {
			t.Errorf("expected nil lastAgentID for contact sender, got %d", *capturedLastAgentID)
		}
	})

	t.Run("attachment is included in last_message JSON when present", func(t *testing.T) {
		var capturedLastMsgJSON []byte

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (int, error) {
				return 1, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
				capturedLastMsgJSON = lastMessageJSON
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		req := SendMessageRequest{
			Text:       strPtr("with attachment"),
			Attachment: json.RawMessage(`{"url":"http://example.com/file.pdf","type":"pdf"}`),
			SenderID:   1,
			SenderType: "user",
		}
		_, err := svc.Send(ctx, req, 1, 10)
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
		if attMap["type"] != "pdf" {
			t.Errorf("expected type pdf, got %v", attMap["type"])
		}
	})

	t.Run("attachment omitted from last_message JSON when absent", func(t *testing.T) {
		var capturedLastMsgJSON []byte

		msgRepo := &mockMessageRepository{
			createFunc: func(ctx context.Context, q DBTX, msg *Message) (int, error) {
				return 1, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
				capturedLastMsgJSON = lastMessageJSON
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		req := SendMessageRequest{
			Text:       strPtr("no attachment"),
			Attachment: nil,
			SenderID:   1,
			SenderType: "user",
		}
		_, err := svc.Send(ctx, req, 1, 10)
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

func TestMessageService_List(t *testing.T) {
	db, err := sql.Open("mock", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	log := logger.NewLogger("debug", "text")
	ctx := context.Background()

	t.Run("pagination defaults when page and limit are zero", func(t *testing.T) {
		var capturedLimit, capturedOffset int

		msgRepo := &mockMessageRepository{
			listFunc: func(ctx context.Context, q DBTX, conversationID int, limit, offset int) ([]*Message, int, error) {
				capturedLimit = limit
				capturedOffset = offset
				return []*Message{}, 0, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		result, err := svc.List(ctx, 1, ListMessageQuery{Page: 0, Limit: 0})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLimit != 50 {
			t.Errorf("expected default limit 50, got %d", capturedLimit)
		}
		if capturedOffset != 0 {
			t.Errorf("expected offset 0, got %d", capturedOffset)
		}
		if result.Meta.PerPage != 50 {
			t.Errorf("expected PerPage 50, got %d", result.Meta.PerPage)
		}
		if result.Meta.CurrentPage != 1 {
			t.Errorf("expected CurrentPage 1, got %d", result.Meta.CurrentPage)
		}
		if result.Meta.Total != 0 {
			t.Errorf("expected Total 0, got %d", result.Meta.Total)
		}
		if result.Meta.LastPage != 0 {
			t.Errorf("expected LastPage 0, got %d", result.Meta.LastPage)
		}
		if result.Meta.From != 0 {
			t.Errorf("expected From 0, got %d", result.Meta.From)
		}
		if result.Meta.To != 0 {
			t.Errorf("expected To 0, got %d", result.Meta.To)
		}
	})

	t.Run("limit defaults to 50 when invalid value is given", func(t *testing.T) {
		var capturedLimit int

		msgRepo := &mockMessageRepository{
			listFunc: func(ctx context.Context, q DBTX, conversationID int, limit, offset int) ([]*Message, int, error) {
				capturedLimit = limit
				return []*Message{}, 0, nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		_, err := svc.List(ctx, 1, ListMessageQuery{Page: 1, Limit: 200})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedLimit != 50 {
			t.Errorf("expected default limit 50, got %d", capturedLimit)
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
		var capturedID int
		var capturedStatus string

		msgRepo := &mockMessageRepository{
			updateStatusFunc: func(ctx context.Context, q DBTX, id int, status string) error {
				capturedID = id
				capturedStatus = status
				return nil
			},
		}
		convRepo := &mockConversationRepository{
			updateLastMessageFunc: func(ctx context.Context, q conversation.DBTX, tenantID int, id int, lastMessageJSON []byte, lastAgentID *int) error {
				return nil
			},
		}

		svc := NewMessageService(msgRepo, convRepo, &mockProfileRepository{}, &mockChannelRepository{}, &mockTenantRepository{}, db, cfg, log, nil)
		if err := svc.UpdateStatus(ctx, 5, "read"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedID != 5 {
			t.Errorf("expected id 5, got %d", capturedID)
		}
		if capturedStatus != "read" {
			t.Errorf("expected status 'read', got %s", capturedStatus)
		}
	})
}
