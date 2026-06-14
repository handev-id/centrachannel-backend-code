package conversation

import (
	"context"
	"database/sql"
	"testing"

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
	listCalled    bool
	listTenantID  int
	listLimit     int
	listOffset    int
	listStatus    string
	listChannelID int
	listAgentID   int
	listSearch    string

	getByIDCalled bool

	createCalled bool
	createConv   *Conversation

	updateStatusCalled   bool
	updateStatusID       int
	updateStatusString   string

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

	listConvs []*Conversation
	listTotal int
	listErr   error

	getByIDConv *Conversation
	getByIDErr  error

	createID  int
	createErr error

	updateStatusErr error

	assignErr   error
	unassignErr error
	markReadErr error
}

func (m *mockConversationRepository) List(_ context.Context, _ DBTX, tenantID int, limit, offset int, status string, channelID, agentID int, search string) ([]*Conversation, int, error) {
	m.listCalled = true
	m.listTenantID = tenantID
	m.listLimit = limit
	m.listOffset = offset
	m.listStatus = status
	m.listChannelID = channelID
	m.listAgentID = agentID
	m.listSearch = search
	return m.listConvs, m.listTotal, m.listErr
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

func (m *mockConversationRepository) UpdateLastMessage(_ context.Context, _ DBTX, _, _ int, _ []byte, _ int) error {
	return nil
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

func TestList(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		mockRepo := &mockConversationRepository{
			listConvs: []*Conversation{{ID: 1, TenantID: 1}},
			listTotal: 1,
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		resp, err := svc.List(context.Background(), ListConversationQuery{}, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if !mockRepo.listCalled {
			t.Error("List was not called")
		}
		if mockRepo.listLimit != 20 {
			t.Errorf("expected default limit 20, got %d", mockRepo.listLimit)
		}
		if mockRepo.listOffset != 0 {
			t.Errorf("expected offset 0, got %d", mockRepo.listOffset)
		}

		if resp.Meta.Total != 1 {
			t.Errorf("expected total 1, got %d", resp.Meta.Total)
		}
		if resp.Meta.PerPage != 20 {
			t.Errorf("expected per_page 20, got %d", resp.Meta.PerPage)
		}
		if resp.Meta.CurrentPage != 1 {
			t.Errorf("expected current_page 1, got %d", resp.Meta.CurrentPage)
		}
		if resp.Meta.LastPage != 1 {
			t.Errorf("expected last_page 1, got %d", resp.Meta.LastPage)
		}
		if resp.Meta.From != 1 {
			t.Errorf("expected from 1, got %d", resp.Meta.From)
		}
		if resp.Meta.To != 1 {
			t.Errorf("expected to 1, got %d", resp.Meta.To)
		}
	})

	t.Run("search_filter", func(t *testing.T) {
		mockRepo := &mockConversationRepository{
			listConvs: []*Conversation{},
			listTotal: 0,
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		q := ListConversationQuery{Page: 1, Limit: 10, Search: "john"}
		_, err := svc.List(context.Background(), q, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if mockRepo.listSearch != "john" {
			t.Errorf("expected search 'john', got %q", mockRepo.listSearch)
		}
	})

	t.Run("status_filter", func(t *testing.T) {
		mockRepo := &mockConversationRepository{
			listConvs: []*Conversation{},
			listTotal: 0,
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		q := ListConversationQuery{Page: 1, Limit: 10, Status: "active"}
		_, err := svc.List(context.Background(), q, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if mockRepo.listStatus != "active" {
			t.Errorf("expected status 'active', got %q", mockRepo.listStatus)
		}
	})

	t.Run("channel_filter", func(t *testing.T) {
		mockRepo := &mockConversationRepository{
			listConvs: []*Conversation{},
			listTotal: 0,
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		q := ListConversationQuery{Page: 1, Limit: 10, ChannelID: 5}
		_, err := svc.List(context.Background(), q, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if mockRepo.listChannelID != 5 {
			t.Errorf("expected channel_id 5, got %d", mockRepo.listChannelID)
		}
	})

	t.Run("agent_filter", func(t *testing.T) {
		mockRepo := &mockConversationRepository{
			listConvs: []*Conversation{},
			listTotal: 0,
		}
		svc := &conversationService{repo: mockRepo, db: nil, cfg: testConfig(), logger: testLogger()}

		q := ListConversationQuery{Page: 1, Limit: 10, AgentID: 3}
		_, err := svc.List(context.Background(), q, testTenant())
		if err != nil {
			t.Fatal(err)
		}

		if mockRepo.listAgentID != 3 {
			t.Errorf("expected agent_id 3, got %d", mockRepo.listAgentID)
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
