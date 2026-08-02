package conversation

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

type mockDB struct {
	beginTxCalled        bool
	execContextCalled    bool
	queryContextCalled   bool
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
	updateStatusID     int
	updateStatusString string

	assignCalled   bool
	assignTenantID int
	assignConvID   int
	assignAgentID  int

	unassignCalled   bool
	unassignTenantID int
	unassignConvID   int

	markReadCalled   bool
	markReadTenantID int
	markReadConvID   int

	listCursorCalled       bool
	listCursorLimit        int
	listCursorLastActivity *time.Time
	listCursorLastID       int
	listCursorConvs        []*Conversation
	listCursorErr          error

	getByIDConv *Conversation
	getByIDErr  error

	createID  int
	createErr error

	updateStatusErr error

	assignErr   error
	unassignErr error
	markReadErr error
}

func (m *mockConversationRepository) GetByID(_ context.Context, _ DBTX, _, _ int) (*Conversation, error) {
	m.getByIDCalled = true
	return m.getByIDConv, m.getByIDErr
}

func (m *mockConversationRepository) Create(_ context.Context, _ DBTX, conv *Conversation) (int, error) {
	m.createCalled = true
	m.createConv = conv
	return m.createID, m.createErr
}

func (m *mockConversationRepository) UpdateStatus(_ context.Context, _ DBTX, tenantID, id int, status string) error {
	m.updateStatusCalled = true
	m.updateStatusID = id
	m.updateStatusString = status
	return m.updateStatusErr
}

func (m *mockConversationRepository) Assign(_ context.Context, _ DBTX, tenantID, id, agentID int) error {
	m.assignCalled = true
	m.assignTenantID = tenantID
	m.assignConvID = id
	m.assignAgentID = agentID
	return m.assignErr
}

func (m *mockConversationRepository) Unassign(_ context.Context, _ DBTX, tenantID, id int) error {
	m.unassignCalled = true
	m.unassignTenantID = tenantID
	m.unassignConvID = id
	return m.unassignErr
}

func (m *mockConversationRepository) GetTotalUnread(_ context.Context, _ DBTX, tenantID int) (int, error) {
	return 0, nil
}

func (m *mockConversationRepository) MarkRead(_ context.Context, _ DBTX, tenantID, id int) error {
	m.markReadCalled = true
	m.markReadTenantID = tenantID
	m.markReadConvID = id
	return m.markReadErr
}

func (m *mockConversationRepository) ListCursor(_ context.Context, _ DBTX, _ int, limit int, _ string, _, _ int, _ string, lastActivity *time.Time, lastID int) ([]*Conversation, error) {
	m.listCursorCalled = true
	m.listCursorLimit = limit
	m.listCursorLastActivity = lastActivity
	m.listCursorLastID = lastID
	return m.listCursorConvs, m.listCursorErr
}

func (m *mockConversationRepository) UpdateLastMessage(_ context.Context, _ DBTX, _, _ int, _ []byte, _ *int) error {
	return nil
}

func (m *mockConversationRepository) FindOpenByProfileAndChannel(_ context.Context, _ DBTX, _, _, _ int) (*Conversation, error) {
	return nil, nil
}

func testConfig() *config.Config {
	return &config.Config{}
}

func testLogger() *logger.Logger {
	return logger.NewLogger("debug", "text")
}

func testTenant() *tenant.Tenant {
	return &tenant.Tenant{ID: 1, Name: "Test Tenant"}
}

func TestListCursor(t *testing.T) {
	t.Run("defaults_and_passes_cursor", func(t *testing.T) {
		activity := time.Date(2026, 8, 2, 10, 0, 0, 0, time.UTC)
		mockRepo := &mockConversationRepository{
			listCursorConvs: []*Conversation{
				{ID: 20, TenantID: 1, LastActivity: &activity},
				{ID: 19, TenantID: 1, LastActivity: &activity},
			},
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		q := ListConversationQuery{Limit: 0, LastID: 20, LastActivity: activity.Format(time.RFC3339)}
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
		if mockRepo.listCursorLastID != 20 {
			t.Errorf("expected lastID 20, got %d", mockRepo.listCursorLastID)
		}
		if mockRepo.listCursorLastActivity == nil || !mockRepo.listCursorLastActivity.Equal(activity) {
			t.Errorf("expected lastActivity %v, got %v", activity, mockRepo.listCursorLastActivity)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 conversations, got %d", len(result))
		}
		if lastID != 19 {
			t.Errorf("expected lastID 19, got %d", lastID)
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
			convs = append(convs, &Conversation{ID: i, TenantID: 1, LastActivity: &activity})
		}
		mockRepo := &mockConversationRepository{listCursorConvs: convs}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, lastID, _, hasMore, err := svc.ListCursor(context.Background(), ListConversationQuery{Limit: 20, LastID: 41, LastActivity: activity.Format(time.RFC3339)}, testTenant())
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 20 {
			t.Errorf("expected 20 conversations, got %d", len(result))
		}
		if !hasMore {
			t.Error("expected hasMore true")
		}
		if lastID != 21 {
			t.Errorf("expected lastID 21 (last kept item), got %d", lastID)
		}
	})

	t.Run("invalid_last_activity", func(t *testing.T) {
		mockRepo := &mockConversationRepository{}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		_, _, _, _, err := svc.ListCursor(context.Background(), ListConversationQuery{Limit: 20, LastID: 10, LastActivity: "not-a-time"}, testTenant())
		if err == nil {
			t.Error("expected error for invalid last_activity")
		}
	})

	t.Run("null_activity_tail_passes_id_only", func(t *testing.T) {
		mockRepo := &mockConversationRepository{
			listCursorConvs: []*Conversation{
				{ID: 30, TenantID: 1}, // LastActivity nil
				{ID: 29, TenantID: 1}, // LastActivity nil
			},
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, lastID, lastActivity, hasMore, err := svc.ListCursor(context.Background(), ListConversationQuery{Limit: 20, LastID: 31, LastActivity: ""}, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if !mockRepo.listCursorCalled {
			t.Error("ListCursor was not called")
		}
		if mockRepo.listCursorLastActivity != nil {
			t.Errorf("expected nil lastActivity for null-activity tail, got %v", mockRepo.listCursorLastActivity)
		}
		if mockRepo.listCursorLastID != 31 {
			t.Errorf("expected lastID 31, got %d", mockRepo.listCursorLastID)
		}

		if len(result) != 2 {
			t.Errorf("expected 2 conversations, got %d", len(result))
		}
		if lastID != 29 {
			t.Errorf("expected lastID 29, got %d", lastID)
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
		mockRepo := &mockConversationRepository{createID: 10}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		req := CreateConversationRequest{ProfileID: 1, ChannelID: 2}
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
		if mockRepo.createConv.ProfileID != 1 {
			t.Errorf("expected profile_id 1, got %d", mockRepo.createConv.ProfileID)
		}
		if mockRepo.createConv.ChannelID != 2 {
			t.Errorf("expected channel_id 2, got %d", mockRepo.createConv.ChannelID)
		}
		if mockRepo.createConv.TenantID != 1 {
			t.Errorf("expected tenant_id 1, got %d", mockRepo.createConv.TenantID)
		}

		if conv.ID != 10 {
			t.Errorf("expected returned id 10, got %d", conv.ID)
		}
		if conv.CreatedAt.IsZero() {
			t.Error("expected non-zero CreatedAt")
		}
		if conv.UpdatedAt.IsZero() {
			t.Error("expected non-zero UpdatedAt")
		}
	})

	t.Run("assigned", func(t *testing.T) {
		mockRepo := &mockConversationRepository{createID: 11}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		req := CreateConversationRequest{ProfileID: 1, ChannelID: 2, AgentID: 7}
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
		if *mockRepo.createConv.AgentID != 7 {
			t.Errorf("expected agent_id 7, got %d", *mockRepo.createConv.AgentID)
		}
		if mockRepo.createConv.ProfileID != 1 {
			t.Errorf("expected profile_id 1, got %d", mockRepo.createConv.ProfileID)
		}
		if mockRepo.createConv.ChannelID != 2 {
			t.Errorf("expected channel_id 2, got %d", mockRepo.createConv.ChannelID)
		}

		if conv.ID != 11 {
			t.Errorf("expected returned id 11, got %d", conv.ID)
		}
	})
}

func TestAssign(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Assign(context.Background(), 1, 42, 7)
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.assignCalled {
		t.Error("Assign was not called on repository")
	}
	if mockRepo.assignTenantID != 1 {
		t.Errorf("expected tenant_id 1, got %d", mockRepo.assignTenantID)
	}
	if mockRepo.assignConvID != 42 {
		t.Errorf("expected conversation id 42, got %d", mockRepo.assignConvID)
	}
	if mockRepo.assignAgentID != 7 {
		t.Errorf("expected agent_id 7, got %d", mockRepo.assignAgentID)
	}
}

func TestUnassign(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Unassign(context.Background(), 1, 42)
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.unassignCalled {
		t.Error("Unassign was not called on repository")
	}
	if mockRepo.unassignTenantID != 1 {
		t.Errorf("expected tenant_id 1, got %d", mockRepo.unassignTenantID)
	}
	if mockRepo.unassignConvID != 42 {
		t.Errorf("expected conversation id 42, got %d", mockRepo.unassignConvID)
	}
}

func TestResolve(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Resolve(context.Background(), 1, 42)
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.updateStatusCalled {
		t.Error("UpdateStatus was not called on repository")
	}
	if mockRepo.updateStatusID != 42 {
		t.Errorf("expected conversation id 42, got %d", mockRepo.updateStatusID)
	}
	if mockRepo.updateStatusString != "resolved" {
		t.Errorf("expected status 'resolved', got %q", mockRepo.updateStatusString)
	}
}

func TestReopen(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.Reopen(context.Background(), 1, 42)
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.updateStatusCalled {
		t.Error("UpdateStatus was not called on repository")
	}
	if mockRepo.updateStatusID != 42 {
		t.Errorf("expected conversation id 42, got %d", mockRepo.updateStatusID)
	}
	if mockRepo.updateStatusString != "unassigned" {
		t.Errorf("expected status 'unassigned', got %q", mockRepo.updateStatusString)
	}
}

func TestMarkRead(t *testing.T) {
	mockRepo := &mockConversationRepository{}
	svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

	err := svc.MarkRead(context.Background(), 1, 42)
	if err != nil {
		t.Fatal(err)
	}

	if !mockRepo.markReadCalled {
		t.Error("MarkRead was not called on repository")
	}
	if mockRepo.markReadTenantID != 1 {
		t.Errorf("expected tenant_id 1, got %d", mockRepo.markReadTenantID)
	}
	if mockRepo.markReadConvID != 42 {
		t.Errorf("expected conversation id 42, got %d", mockRepo.markReadConvID)
	}
}
