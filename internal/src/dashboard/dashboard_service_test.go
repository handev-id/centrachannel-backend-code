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
	getTotalContactsFunc       func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getActiveConversationsFunc func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getResolvedTodayFunc       func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getUnassignedCountFunc     func(ctx context.Context, q DBTX, tenantID int) (int, error)
	getConversationChartFunc   func(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error)
}

func (m *mockDashboardRepository) GetTotalContacts(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getTotalContactsFunc(ctx, q, tenantID)
}

func (m *mockDashboardRepository) GetActiveConversations(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getActiveConversationsFunc(ctx, q, tenantID)
}

func (m *mockDashboardRepository) GetResolvedToday(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getResolvedTodayFunc(ctx, q, tenantID)
}

func (m *mockDashboardRepository) GetUnassignedCount(ctx context.Context, q DBTX, tenantID int) (int, error) {
	return m.getUnassignedCountFunc(ctx, q, tenantID)
}

func (m *mockDashboardRepository) GetConversationChart(ctx context.Context, q DBTX, tenantID int, days int) ([]ChartDataPoint, error) {
	return m.getConversationChartFunc(ctx, q, tenantID, days)
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

func TestDashboardService_GetStats(t *testing.T) {
	t.Run("returns stats with correct counts and uses tenant_id", func(t *testing.T) {
		var capturedTenantID int
		repo := &mockDashboardRepository{
			getTotalContactsFunc: func(ctx context.Context, q DBTX, tenantID int) (int, error) {
				capturedTenantID = tenantID
				return 100, nil
			},
			getActiveConversationsFunc: func(ctx context.Context, q DBTX, tenantID int) (int, error) {
				return 25, nil
			},
			getResolvedTodayFunc: func(ctx context.Context, q DBTX, tenantID int) (int, error) {
				return 10, nil
			},
			getUnassignedCountFunc: func(ctx context.Context, q DBTX, tenantID int) (int, error) {
				return 5, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		result, err := svc.GetStats(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if result.TotalContacts != 100 {
			t.Errorf("expected TotalContacts 100, got %d", result.TotalContacts)
		}
		if result.ActiveConversations != 25 {
			t.Errorf("expected ActiveConversations 25, got %d", result.ActiveConversations)
		}
		if result.ResolvedToday != 10 {
			t.Errorf("expected ResolvedToday 10, got %d", result.ResolvedToday)
		}
		if result.UnassignedCount != 5 {
			t.Errorf("expected UnassignedCount 5, got %d", result.UnassignedCount)
		}
	})
}

func TestDashboardService_GetChart(t *testing.T) {
	t.Run("returns chart data with labels and values and uses tenant_id", func(t *testing.T) {
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
