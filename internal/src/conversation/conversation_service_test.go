package conversation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

type mockDB struct {
	beginTxCalled         bool
	execContextCalled     bool
	queryContextCalled    bool
	queryRowContextCalled bool
}

func (m *mockDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	m.execContextCalled = true
	return nil, nil
}

func (m *mockDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	m.queryContextCalled = true
	return nil, nil
}

func (m *mockDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	m.queryRowContextCalled = true
	return nil
}

func (m *mockDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	m.beginTxCalled = true
	return nil, nil
}

type mockConversationRepository struct {
	getByIDCalled bool

	createCalled bool
	createConv   *Conversation

	updateStatusCalled bool
	updateStatusID     uuid.UUID
	updateStatusString string

	assignCalled   bool
	assignTenantID uuid.UUID
	assignConvID   uuid.UUID
	assignAgentID  uuid.UUID

	unassignCalled   bool
	unassignTenantID uuid.UUID
	unassignConvID   uuid.UUID

	markReadCalled   bool
	markReadTenantID uuid.UUID
	markReadConvID   uuid.UUID

	listCursorCalled       bool
	listCursorLimit        int
	listCursorLastActivity *time.Time
	listCursorLastID       uuid.UUID
	listCursorConvs        []*Conversation
	listCursorErr          error

	getByIDConv *Conversation
	getByIDErr  error

	createID  uuid.UUID
	createErr error

	updateStatusErr error

	assignErr   error
	unassignErr error
	markReadErr error
}

func (m *mockConversationRepository) GetByID(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID) (*Conversation, error) {
	m.getByIDCalled = true
	return m.getByIDConv, m.getByIDErr
}

func (m *mockConversationRepository) Create(_ context.Context, _ DBTX, conv *Conversation) (uuid.UUID, error) {
	m.createCalled = true
	m.createConv = conv
	return m.createID, m.createErr
}

func (m *mockConversationRepository) UpdateStatus(_ context.Context, _ DBTX, tenantID uuid.UUID, id uuid.UUID, status string) error {
	m.updateStatusCalled = true
	m.updateStatusID = id
	m.updateStatusString = status
	return m.updateStatusErr
}

func (m *mockConversationRepository) Assign(_ context.Context, _ DBTX, tenantID uuid.UUID, id uuid.UUID, agentID uuid.UUID) error {
	m.assignCalled = true
	m.assignTenantID = tenantID
	m.assignConvID = id
	m.assignAgentID = agentID
	return m.assignErr
}

func (m *mockConversationRepository) Unassign(_ context.Context, _ DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	m.unassignCalled = true
	m.unassignTenantID = tenantID
	m.unassignConvID = id
	return m.unassignErr
}

func (m *mockConversationRepository) GetTotalUnread(_ context.Context, _ DBTX, tenantID uuid.UUID) (int, error) {
	return 0, nil
}

func (m *mockConversationRepository) MarkRead(_ context.Context, _ DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	m.markReadCalled = true
	m.markReadTenantID = tenantID
	m.markReadConvID = id
	return m.markReadErr
}

func (m *mockConversationRepository) ListCursor(_ context.Context, _ DBTX, _ uuid.UUID, limit int, _ string, _ uuid.UUID, _ uuid.UUID, _ string, lastActivity *time.Time, lastID uuid.UUID) ([]*Conversation, error) {
	m.listCursorCalled = true
	m.listCursorLimit = limit
	m.listCursorLastActivity = lastActivity
	m.listCursorLastID = lastID
	return m.listCursorConvs, m.listCursorErr
}

func (m *mockConversationRepository) UpdateLastMessage(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID, _ []byte, _ *uuid.UUID) error {
	return nil
}

func (m *mockConversationRepository) FindOpenByProfileAndChannel(_ context.Context, _ DBTX, _ uuid.UUID, _ uuid.UUID, _ uuid.UUID) (*Conversation, error) {
	return nil, nil
}

func testConfig() *config.Config {
	return &config.Config{}
}

func testLogger() *logger.Logger {
	return logger.NewLogger("debug", "text")
}

func testTenant() *tenant.Tenant {
	return &tenant.Tenant{ID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), Name: "Test Tenant"}
}

func TestListCursor(t *testing.T) {
	t.Run("defaults_and_passes_cursor", func(t *testing.T) {
		activity := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
		mockRepo := &mockConversationRepository{
			listCursorConvs: []*Conversation{
				{ID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), LastActivity: &activity},
				{ID: uuid.MustParse("00000000-0000-4000-8000-000000000013"), TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), LastActivity: &activity},
			},
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		q := ListConversationQuery{Limit: 0, LastID: uuid.MustParse("00000000-0000-4000-8000-000000000014"), LastActivity: activity.Format(time.RFC3339)}
		result, lastID, lastActivity, hasMore, err := svc.ListCursor(context.Background(), q, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if !mockRepo.listCursorCalled {
			t.Error("ListCursor was not called")
		}
		if mockRepo.listCursorLimit != 21 {
			t.Errorf("expected limit+1 = 21, got %d", mockRepo.listCursorLimit)
		}
		if mockRepo.listCursorLastID != uuid.MustParse("00000000-0000-4000-8000-000000000014") {
			t.Errorf("expected lastID 20, got %s", mockRepo.listCursorLastID)
		}
		if mockRepo.listCursorLastActivity == nil || !mockRepo.listCursorLastActivity.Equal(activity) {
			t.Errorf("expected lastActivity %v, got %v", activity, mockRepo.listCursorLastActivity)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 conversations, got %d", len(result))
		}
		if lastID != uuid.MustParse("00000000-0000-4000-8000-000000000013") {
			t.Errorf("expected lastID 19, got %s", lastID)
		}
		if lastActivity != activity.Format(time.RFC3339) {
			t.Errorf("expected lastActivity %s, got %s", activity.Format(time.RFC3339), lastActivity)
		}
		if hasMore {
			t.Error("expected hasMore false")
		}
	})

	t.Run("has_more_trims_extra_item", func(t *testing.T) {
		activity := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
		convs := make([]*Conversation, 0, 21)
		for i := 40; i >= 20; i-- {
			convs = append(convs, &Conversation{ID: uuid.UUID{6: 0x40, 8: 0x80, 15: byte(i)}, TenantID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), LastActivity: &activity})
		}
		mockRepo := &mockConversationRepository{listCursorConvs: convs}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, lastID, _, hasMore, err := svc.ListCursor(context.Background(), ListConversationQuery{Limit: 20, LastID: uuid.MustParse("00000000-0000-4000-8000-000000000029"), LastActivity: activity.Format(time.RFC3339)}, testTenant())
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 20 {
			t.Errorf("expected 20 conversations, got %d", len(result))
		}
		if !hasMore {
			t.Error("expected hasMore true")
		}
		if lastID != uuid.MustParse("00000000-0000-4000-8000-000000000015") {
			t.Errorf("expected lastID 21 (last kept item), got %s", lastID)
		}
	})

	t.Run("invalid_last_activity", func(t *testing.T) {
		mockRepo := &mockConversationRepository{}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		_, _, _, _, err := svc.ListCursor(context.Background(), ListConversationQuery{Limit: 20, LastID: uuid.MustParse("00000000-0000-4000-8000-00000000000a"), LastActivity: "not-a-time"}, testTenant())
		if err == nil {
			t.Error("expected error for invalid last_activity")
		}
	})

	t.Run("null_activity_tail_passes_id_only", func(t *testing.T) {
		mockRepo := &mockConversationRepository{
			listCursorConvs: []*Conversation{
				{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001e"), TenantID: uuid.MustParse( // LastActivity nil
					"00000000-0000-4000-8000-000000000001")},
				{ID: uuid.MustParse("00000000-0000-4000-8000-00000000001d"), TenantID: uuid.MustParse( // LastActivity nil
					"00000000-0000-4000-8000-000000000001")},
			},
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, lastID, lastActivity, hasMore, err := svc.ListCursor(context.Background(), ListConversationQuery{Limit: 20, LastID: uuid.MustParse("00000000-0000-4000-8000-00000000001f"), LastActivity: ""}, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if !mockRepo.listCursorCalled {
			t.Error("ListCursor was not called")
		}
		if mockRepo.listCursorLastActivity != nil {
			t.Errorf("expected nil lastActivity for null-activity tail, got %v", mockRepo.listCursorLastActivity)
		}
		if mockRepo.listCursorLastID != uuid.MustParse("00000000-0000-4000-8000-00000000001f") {
			t.Errorf("expected lastID 31, got %s", mockRepo.listCursorLastID)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 conversations, got %d", len(result))
		}
		if lastID != uuid.MustParse("00000000-0000-4000-8000-00000000001d") {
			t.Errorf("expected lastID 29, got %s", lastID)
		}
		if lastActivity != "" {
			t.Errorf("expected empty lastActivity, got %q", lastActivity)
		}
		if hasMore {
			t.Error("expected hasMore false")
		}
	})
}

func TestCreate(t *testing.T) {
	t.Run("unassigned", func(t *testing.T) {
		mockRepo := &mockConversationRepository{createID: uuid.MustParse("00000000-0000-4000-8000-00000000000a")}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		req := CreateConversationRequest{ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), ChannelID: uuid.MustParse("00000000-0000-4000-8000-000000000002")}
		conv, err := svc.Create(context.Background(), req, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if !mockRepo.createCalled {
			t.Error("Create was not called")
		}
		if mockRepo.createConv.Status != "unassigned" {
			t.Errorf("expected status 'unassigned', got %q", mockRepo.createConv.Status)
		}
		if mockRepo.createConv.AgentID != nil {
			t.Error("expected nil agent_id for unassigned conversation")
		}
		if mockRepo.createConv.ProfileID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected profile_id 1, got %s", mockRepo.createConv.ProfileID)
		}
		if mockRepo.createConv.ChannelID != uuid.MustParse("00000000-0000-4000-8000-000000000002") {
			t.Errorf("expected channel_id 2, got %s", mockRepo.createConv.ChannelID)
		}
		if mockRepo.createConv.TenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected tenant_id 1, got %s", mockRepo.createConv.TenantID)
		}

		if conv.ID != uuid.MustParse("00000000-0000-4000-8000-00000000000a") {
			t.Errorf("expected returned id 10, got %s", conv.ID)
		}
		if conv.CreatedAt.IsZero() {
			t.Error("expected non-zero CreatedAt")
		}
		if conv.UpdatedAt.IsZero() {
			t.Error("expected non-zero UpdatedAt")
		}
	})

	t.Run("assigned", func(t *testing.T) {
		mockRepo := &mockConversationRepository{createID: uuid.MustParse("00000000-0000-4000-8000-00000000000b")}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		req := CreateConversationRequest{ProfileID: uuid.MustParse("00000000-0000-4000-8000-000000000001"), ChannelID: uuid.MustParse("00000000-0000-4000-8000-000000000002"), AgentID: uuid.MustParse("00000000-0000-4000-8000-000000000007")}
		conv, err := svc.Create(context.Background(), req, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if !mockRepo.createCalled {
			t.Error("Create was not called")
		}
		if mockRepo.createConv.Status != "assigned" {
			t.Errorf("expected status 'assigned', got %q", mockRepo.createConv.Status)
		}
		if mockRepo.createConv.AgentID == nil {
			t.Fatal("expected non-nil agent_id for assigned conversation")
		}
		if *mockRepo.createConv.AgentID != uuid.MustParse("00000000-0000-4000-8000-000000000007") {
			t.Errorf("expected agent_id 7, got %s", *mockRepo.createConv.AgentID)
		}
		if mockRepo.createConv.ProfileID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
			t.Errorf("expected profile_id 1, got %s", mockRepo.createConv.ProfileID)
		}
		if mockRepo.createConv.ChannelID != uuid.MustParse("00000000-0000-4000-8000-000000000002") {
			t.Errorf("expected channel_id 2, got %s", mockRepo.createConv.ChannelID)
		}

		if conv.ID != uuid.MustParse("00000000-0000-4000-8000-00000000000b") {
			t.Errorf("expected returned id 11, got %s", conv.ID)
		}
	})
}

func TestAssign(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Assign(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000002a"), uuid.MustParse("00000000-0000-4000-8000-000000000007"))
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.assignCalled {
		t.Error("Assign was not called on repository")
	}
	if mockRepo.assignTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected tenant_id 1, got %s", mockRepo.assignTenantID)
	}
	if mockRepo.assignConvID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
		t.Errorf("expected conversation id 42, got %s", mockRepo.assignConvID)
	}
	if mockRepo.assignAgentID != uuid.MustParse("00000000-0000-4000-8000-000000000007") {
		t.Errorf("expected agent_id 7, got %s", mockRepo.assignAgentID)
	}
}

func TestUnassign(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Unassign(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000002a"))
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.unassignCalled {
		t.Error("Unassign was not called on repository")
	}
	if mockRepo.unassignTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected tenant_id 1, got %s", mockRepo.unassignTenantID)
	}
	if mockRepo.unassignConvID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
		t.Errorf("expected conversation id 42, got %s", mockRepo.unassignConvID)
	}
}

func TestResolve(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Resolve(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000002a"))
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.updateStatusCalled {
		t.Error("UpdateStatus was not called on repository")
	}
	if mockRepo.updateStatusID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
		t.Errorf("expected conversation id 42, got %s", mockRepo.updateStatusID)
	}
	if mockRepo.updateStatusString != "resolved" {
		t.Errorf("expected status 'resolved', got %q", mockRepo.updateStatusString)
	}
}

func TestReopen(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Reopen(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000002a"))
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.updateStatusCalled {
		t.Error("UpdateStatus was not called on repository")
	}
	if mockRepo.updateStatusID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
		t.Errorf("expected conversation id 42, got %s", mockRepo.updateStatusID)
	}
	if mockRepo.updateStatusString != "unassigned" {
		t.Errorf("expected status 'unassigned', got %q", mockRepo.updateStatusString)
	}
}

func TestMarkRead(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.MarkRead(context.Background(), uuid.MustParse("00000000-0000-4000-8000-000000000001"), uuid.MustParse("00000000-0000-4000-8000-00000000002a"))
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.markReadCalled {
		t.Error("MarkRead was not called on repository")
	}
	if mockRepo.markReadTenantID != uuid.MustParse("00000000-0000-4000-8000-000000000001") {
		t.Errorf("expected tenant_id 1, got %s", mockRepo.markReadTenantID)
	}
	if mockRepo.markReadConvID != uuid.MustParse("00000000-0000-4000-8000-00000000002a") {
		t.Errorf("expected conversation id 42, got %s", mockRepo.markReadConvID)
	}
}
