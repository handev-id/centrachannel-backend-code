package dashboard

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"testing"
)

var mockDriverInst = &mockDriver{}

func init() {
	for _, name := range sql.Drivers() {
		if name == "mock" {
			return
		}
	}
	sql.Register("mock", mockDriverInst)
}

type mockDriver struct {
	conn *mockConn
}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return d.conn, nil
}

type mockConn struct {
	beginTxFunc func(ctx context.Context, opts driver.TxOptions) (driver.Tx, error)
	execFunc    func(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error)
	queryFunc   func(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error)
	prepareFunc func(query string) (driver.Stmt, error)
	closeFunc   func() error
	beginFunc   func() (driver.Tx, error)
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	if c.prepareFunc != nil {
		return c.prepareFunc(query)
	}
	return nil, fmt.Errorf("Prepare not implemented")
}

func (c *mockConn) Close() error {
	if c.closeFunc != nil {
		return c.closeFunc()
	}
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	if c.beginFunc != nil {
		return c.beginFunc()
	}
	return nil, fmt.Errorf("Begin not implemented")
}

func (c *mockConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if c.beginTxFunc != nil {
		return c.beginTxFunc(ctx, opts)
	}
	return nil, fmt.Errorf("BeginTx not implemented")
}

func (c *mockConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if c.execFunc != nil {
		return c.execFunc(ctx, query, args)
	}
	return nil, fmt.Errorf("ExecContext not implemented")
}

func (c *mockConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if c.queryFunc != nil {
		return c.queryFunc(ctx, query, args)
	}
	return nil, fmt.Errorf("QueryContext not implemented")
}

var (
	_ driver.Conn           = (*mockConn)(nil)
	_ driver.ConnBeginTx    = (*mockConn)(nil)
	_ driver.ExecerContext  = (*mockConn)(nil)
	_ driver.QueryerContext = (*mockConn)(nil)
)

type mockTx struct {
	commitFunc   func() error
	rollbackFunc func() error
}

func (t *mockTx) Commit() error {
	if t.commitFunc != nil {
		return t.commitFunc()
	}
	return nil
}

func (t *mockTx) Rollback() error {
	if t.rollbackFunc != nil {
		return t.rollbackFunc()
	}
	return nil
}

var _ driver.Tx = (*mockTx)(nil)

type mockRows struct {
	columns []string
	data    [][]driver.Value
	pos     int
}

func (r *mockRows) Columns() []string { return r.columns }
func (r *mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

var _ driver.Rows = (*mockRows)(nil)

type mockResult struct {
	lastInsertID int64
	rowsAffected int64
}

func (r *mockResult) LastInsertId() (int64, error) { return r.lastInsertID, nil }
func (r *mockResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

var _ driver.Result = (*mockResult)(nil)

type mockDashboardRepository struct {
	getTotalContactsFunc            func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getNewContactsFunc              func(ctx context.Context, q DBTX, tenantID int, days int) (int, error)
	getTotalConversationsFunc       func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getActiveConversationsFunc      func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getNewConversationsFunc         func(ctx context.Context, q DBTX, tenantID int, days int) (int, error)
	getResolvedTodayFunc            func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getUnassignedCountFunc          func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getTotalUnreadConversationsFunc func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getTotalUnreadMessagesFunc      func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getTotalMessagesFunc            func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getMessagesTodayFunc            func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getTotalCampaignsFunc           func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getConnectedWhatsappDevicesFunc func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getConversationsByStatusFunc    func(ctx context.Context, q DBTX, tenantID int) (*ConversationsByStatus, error)
	getConversationsByChannelFunc   func(ctx context.Context, q DBTX, tenantID int) ([]ConversationsByChannel, error)
	getRecentConversationsFunc      func(ctx context.Context, q DBTX, tenantID int, limit int) ([]RecentConversation, error)
	getConversationChartFunc        func(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error)
	getContactsByStatusFunc         func(ctx context.Context, q DBTX, tenantID int) (*ContactsByStatus, error)
	getMessagesBySenderTypeFunc     func(ctx context.Context, q DBTX, tenantID int) (*MessagesBySenderType, error)
	getMessagesByDateFunc           func(ctx context.Context, q DBTX, tenantID int, days int) ([]MessagesByDate, error)
	getCampaignsByStatusFunc        func(ctx context.Context, q DBTX, tenantID int) (map[string]int, error)
	getAgentWorkloadFunc            func(ctx context.Context, q DBTX, tenantID int) ([]AgentWorkload, error)
}

func (m *mockDashboardRepository) GetTotalContacts(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getTotalContactsFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetNewContacts(ctx context.Context, q DBTX, tenantID int, days int) (int, error) {
	return m.getNewContactsFunc(ctx, q, tenantID, days)
}
func (m *mockDashboardRepository) GetTotalConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getTotalConversationsFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetActiveConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getActiveConversationsFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetNewConversations(ctx context.Context, q DBTX, tenantID int, days int) (int, error) {
	return m.getNewConversationsFunc(ctx, q, tenantID, days)
}
func (m *mockDashboardRepository) GetResolvedToday(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getResolvedTodayFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetUnassignedCount(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getUnassignedCountFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetTotalUnreadConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getTotalUnreadConversationsFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetTotalUnreadMessages(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getTotalUnreadMessagesFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetTotalMessages(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getTotalMessagesFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetMessagesToday(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getMessagesTodayFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetTotalCampaigns(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getTotalCampaignsFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetConnectedWhatsappDevices(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getConnectedWhatsappDevicesFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetConversationsByStatus(ctx context.Context, q DBTX, tenantID int) (*ConversationsByStatus, error) {
	return m.getConversationsByStatusFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetConversationsByChannel(ctx context.Context, q DBTX, tenantID int) ([]ConversationsByChannel, error) {
	return m.getConversationsByChannelFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetRecentConversations(ctx context.Context, q DBTX, tenantID int, limit int) ([]RecentConversation, error) {
	return m.getRecentConversationsFunc(ctx, q, tenantID, limit)
}
func (m *mockDashboardRepository) GetConversationChart(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error) {
	return m.getConversationChartFunc(ctx, q, tenantID, days)
}
func (m *mockDashboardRepository) GetContactsByStatus(ctx context.Context, q DBTX, tenantID int) (*ContactsByStatus, error) {
	return m.getContactsByStatusFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetMessagesBySenderType(ctx context.Context, q DBTX, tenantID int) (*MessagesBySenderType, error) {
	return m.getMessagesBySenderTypeFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetMessagesByDate(ctx context.Context, q DBTX, tenantID int, days int) ([]MessagesByDate, error) {
	return m.getMessagesByDateFunc(ctx, q, tenantID, days)
}
func (m *mockDashboardRepository) GetCampaignsByStatus(ctx context.Context, q DBTX, tenantID int) (map[string]int, error) {
	return m.getCampaignsByStatusFunc(ctx, q, tenantID)
}
func (m *mockDashboardRepository) GetAgentWorkload(ctx context.Context, q DBTX, tenantID int) ([]AgentWorkload, error) {
	return m.getAgentWorkloadFunc(ctx, q, tenantID)
}

func newMockDB(conn *mockConn) *sql.DB {
	mockDriverInst.conn = conn
	db, err := sql.Open("mock", "")
	if err != nil {
		panic(fmt.Sprintf("failed to open mock db: %v", err))
	}
	db.SetMaxOpenConns(1)
	return db
}

func newService(repo DashboardRepository, db *sql.DB) DashboardService {
	return NewDashboardService(repo, db)
}

func defaultMockRepo() *mockDashboardRepository {
	return &mockDashboardRepository{
		getTotalContactsFunc:            func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 100, nil },
		getNewContactsFunc:              func(ctx context.Context, q DBTX, tenantID int, days int) (int, error) { return 10, nil },
		getTotalConversationsFunc:       func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 200, nil },
		getActiveConversationsFunc:      func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 50, nil },
		getNewConversationsFunc:         func(ctx context.Context, q DBTX, tenantID int, days int) (int, error) { return 20, nil },
		getResolvedTodayFunc:            func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 15, nil },
		getUnassignedCountFunc:          func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 10, nil },
		getTotalUnreadConversationsFunc: func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 30, nil },
		getTotalUnreadMessagesFunc:      func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 75, nil },
		getTotalMessagesFunc:            func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 5000, nil },
		getMessagesTodayFunc:            func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 45, nil },
		getTotalCampaignsFunc:           func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 12, nil },
		getConnectedWhatsappDevicesFunc: func(ctx context.Context, q DBTX, tenantID int) (int, error) { return 2, nil },
		getConversationsByStatusFunc: func(ctx context.Context, q DBTX, tenantID int) (*ConversationsByStatus, error) {
			return &ConversationsByStatus{Unassigned: 10, Assigned: 40, Resolved: 150}, nil
		},
		getConversationsByChannelFunc: func(ctx context.Context, q DBTX, tenantID int) ([]ConversationsByChannel, error) {
			return []ConversationsByChannel{
				{ChannelID: 1, ChannelName: "whatsapp", Total: 150, Unread: 60},
				{ChannelID: 2, ChannelName: "facebook", Total: 50, Unread: 15},
			}, nil
		},
		getRecentConversationsFunc: func(ctx context.Context, q DBTX, tenantID int, limit int) ([]RecentConversation, error) {
			return []RecentConversation{
				{ID: 1, Status: "assigned", UnreadCount: 2},
				{ID: 2, Status: "unassigned", UnreadCount: 0},
			}, nil
		},
		getConversationChartFunc: func(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error) {
			return []ChartDataPoint{
				{Date: "2026-06-15", Count: 5},
				{Date: "2026-06-16", Count: 8},
			}, nil
		},
		getContactsByStatusFunc: func(ctx context.Context, q DBTX, tenantID int) (*ContactsByStatus, error) {
			return &ContactsByStatus{Individual: 80, Institution: 20}, nil
		},
		getMessagesBySenderTypeFunc: func(ctx context.Context, q DBTX, tenantID int) (*MessagesBySenderType, error) {
			return &MessagesBySenderType{Contact: 3000, User: 2000}, nil
		},
		getMessagesByDateFunc: func(ctx context.Context, q DBTX, tenantID int, days int) ([]MessagesByDate, error) {
			return []MessagesByDate{
				{Date: "2026-06-15", Count: 50},
				{Date: "2026-06-16", Count: 45},
			}, nil
		},
		getCampaignsByStatusFunc: func(ctx context.Context, q DBTX, tenantID int) (map[string]int, error) {
			return map[string]int{"draft": 3, "inprogress": 2, "completed": 7}, nil
		},
		getAgentWorkloadFunc: func(ctx context.Context, q DBTX, tenantID int) ([]AgentWorkload, error) {
			return []AgentWorkload{
				{ID: 1, FirstName: "Alice", OngoingConversations: 12},
				{ID: 2, FirstName: "Bob", OngoingConversations: 8},
			}, nil
		},
	}
}

func TestDashboardService_GetStats(t *testing.T) {
	t.Run("returns complete stats response with correct values and uses tenant_id", func(t *testing.T) {
		var capturedTenantID int
		var capturedDays int

		repo := defaultMockRepo()
		repo.getTotalContactsFunc = func(ctx context.Context, q DBTX, tenantID int) (int, error) {
			capturedTenantID = tenantID
			return 100, nil
		}
		repo.getNewContactsFunc = func(ctx context.Context, q DBTX, tenantID int, days int) (int, error) {
			capturedDays = days
			return 10, nil
		}

		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		result, err := svc.GetStats(context.Background(), 1, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if capturedDays != 30 {
			t.Errorf("expected days 30, got %d", capturedDays)
		}

		if result.Summary.TotalContacts != 100 {
			t.Errorf("expected TotalContacts 100, got %d", result.Summary.TotalContacts)
		}
		if result.Summary.NewContacts != 10 {
			t.Errorf("expected NewContacts 10, got %d", result.Summary.NewContacts)
		}
		if result.Summary.TotalConversations != 200 {
			t.Errorf("expected TotalConversations 200, got %d", result.Summary.TotalConversations)
		}
		if result.Summary.ActiveConversations != 50 {
			t.Errorf("expected ActiveConversations 50, got %d", result.Summary.ActiveConversations)
		}
		if result.Summary.NewConversations != 20 {
			t.Errorf("expected NewConversations 20, got %d", result.Summary.NewConversations)
		}
		if result.Summary.ResolvedToday != 15 {
			t.Errorf("expected ResolvedToday 15, got %d", result.Summary.ResolvedToday)
		}
		if result.Summary.TotalUnreadConversations != 30 {
			t.Errorf("expected TotalUnreadConversations 30, got %d", result.Summary.TotalUnreadConversations)
		}
		if result.Summary.TotalUnreadMessages != 75 {
			t.Errorf("expected TotalUnreadMessages 75, got %d", result.Summary.TotalUnreadMessages)
		}
		if result.Summary.TotalMessages != 5000 {
			t.Errorf("expected TotalMessages 5000, got %d", result.Summary.TotalMessages)
		}
		if result.Summary.MessagesToday != 45 {
			t.Errorf("expected MessagesToday 45, got %d", result.Summary.MessagesToday)
		}
		if result.Summary.TotalCampaigns != 12 {
			t.Errorf("expected TotalCampaigns 12, got %d", result.Summary.TotalCampaigns)
		}
		if result.Summary.ConnectedWhatsappDevices != 2 {
			t.Errorf("expected ConnectedWhatsappDevices 2, got %d", result.Summary.ConnectedWhatsappDevices)
		}

		if result.Conversations.ByStatus.Unassigned != 10 {
			t.Errorf("expected Unassigned 10, got %d", result.Conversations.ByStatus.Unassigned)
		}
		if result.Conversations.ByStatus.Assigned != 40 {
			t.Errorf("expected Assigned 40, got %d", result.Conversations.ByStatus.Assigned)
		}
		if result.Conversations.ByStatus.Resolved != 150 {
			t.Errorf("expected Resolved 150, got %d", result.Conversations.ByStatus.Resolved)
		}

		if len(result.Conversations.ByChannel) != 2 {
			t.Fatalf("expected 2 channels, got %d", len(result.Conversations.ByChannel))
		}
		if result.Conversations.ByChannel[0].ChannelName != "whatsapp" {
			t.Errorf("expected channel name 'whatsapp', got %q", result.Conversations.ByChannel[0].ChannelName)
		}

		if len(result.Conversations.Recent) != 2 {
			t.Errorf("expected 2 recent conversations, got %d", len(result.Conversations.Recent))
		}
		if len(result.Conversations.Trend) != 2 {
			t.Errorf("expected 2 trend data points, got %d", len(result.Conversations.Trend))
		}

		if result.Contacts.ByStatus.Individual != 80 {
			t.Errorf("expected Individual 80, got %d", result.Contacts.ByStatus.Individual)
		}
		if result.Contacts.ByStatus.Institution != 20 {
			t.Errorf("expected Institution 20, got %d", result.Contacts.ByStatus.Institution)
		}
		if result.Contacts.NewContacts != 10 {
			t.Errorf("expected NewContacts 10, got %d", result.Contacts.NewContacts)
		}

		if result.Messages.Total != 5000 {
			t.Errorf("expected Messages.Total 5000, got %d", result.Messages.Total)
		}
		if result.Messages.Today != 45 {
			t.Errorf("expected Messages.Today 45, got %d", result.Messages.Today)
		}
		if result.Messages.BySenderType.Contact != 3000 {
			t.Errorf("expected Contact messages 3000, got %d", result.Messages.BySenderType.Contact)
		}
		if result.Messages.BySenderType.User != 2000 {
			t.Errorf("expected User messages 2000, got %d", result.Messages.BySenderType.User)
		}
		if len(result.Messages.ByDate) != 2 {
			t.Errorf("expected 2 message date points, got %d", len(result.Messages.ByDate))
		}

		if result.Campaigns.Total != 12 {
			t.Errorf("expected TotalCampaigns 12, got %d", result.Campaigns.Total)
		}
		if result.Campaigns.ByStatus["draft"] != 3 {
			t.Errorf("expected draft campaigns 3, got %d", result.Campaigns.ByStatus["draft"])
		}
		if result.Campaigns.ByStatus["completed"] != 7 {
			t.Errorf("expected completed campaigns 7, got %d", result.Campaigns.ByStatus["completed"])
		}

		if len(result.Agents) != 2 {
			t.Fatalf("expected 2 agents, got %d", len(result.Agents))
		}
		if result.Agents[0].FirstName != "Alice" {
			t.Errorf("expected first agent Alice, got %s", result.Agents[0].FirstName)
		}
		if result.Agents[0].OngoingConversations != 12 {
			t.Errorf("expected Alice ongoing 12, got %d", result.Agents[0].OngoingConversations)
		}
	})

	t.Run("defaults days to 30 when out of range", func(t *testing.T) {
		var capturedDays int

		repo := defaultMockRepo()
		repo.getNewContactsFunc = func(ctx context.Context, q DBTX, tenantID int, days int) (int, error) {
			capturedDays = days
			return 0, nil
		}

		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		_, err := svc.GetStats(context.Background(), 1, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedDays != 30 {
			t.Errorf("expected days defaulted to 30, got %d", capturedDays)
		}
	})

	t.Run("returns error when repository fails", func(t *testing.T) {
		repo := defaultMockRepo()
		repo.getTotalContactsFunc = func(ctx context.Context, q DBTX, tenantID int) (int, error) {
			return 0, fmt.Errorf("db error")
		}

		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		_, err := svc.GetStats(context.Background(), 1, 30)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestDashboardService_GetChart(t *testing.T) {
	t.Run("returns chart data with correct values and uses tenant_id", func(t *testing.T) {
		var capturedTenantID, capturedDays int
		data := []ChartDataPoint{
			{Date: "2026-06-10", Count: 5},
			{Date: "2026-06-11", Count: 8},
		}
		repo := &mockDashboardRepository{
			getConversationChartFunc: func(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error) {
				capturedTenantID = tenantID
				capturedDays = days
				return data, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		result, err := svc.GetChart(context.Background(), 1, 30)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if capturedDays != 30 {
			t.Errorf("expected days 30, got %d", capturedDays)
		}
		if len(result.Data) != 2 {
			t.Fatalf("expected 2 data points, got %d", len(result.Data))
		}
		if result.Data[0].Date != "2026-06-10" {
			t.Errorf("expected date '2026-06-10', got %q", result.Data[0].Date)
		}
		if result.Data[0].Count != 5 {
			t.Errorf("expected Count 5, got %d", result.Data[0].Count)
		}
		if result.Data[1].Date != "2026-06-11" {
			t.Errorf("expected date '2026-06-11', got %q", result.Data[1].Date)
		}
		if result.Data[1].Count != 8 {
			t.Errorf("expected Count 8, got %d", result.Data[1].Count)
		}
	})

	t.Run("defaults to 30 days when out of range", func(t *testing.T) {
		var capturedDays int
		repo := &mockDashboardRepository{
			getConversationChartFunc: func(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error) {
				capturedDays = days
				return []ChartDataPoint{}, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		_, err := svc.GetChart(context.Background(), 1, 0)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedDays != 30 {
			t.Errorf("expected days defaulted to 30, got %d", capturedDays)
		}
	})
}
