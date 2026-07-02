package whatsapp_device

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"testing"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
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

type mockWhatsAppDeviceRepository struct {
	listFunc          func(ctx context.Context, q DBTX, tenantID int) ([]WhatsAppDevice, error)
	getByIDFunc       func(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error)
	createFunc        func(ctx context.Context, q DBTX, device *WhatsAppDevice) (int, error)
	updateFunc        func(ctx context.Context, q DBTX, tenantID int, id int, device *WhatsAppDevice) error
	deleteFunc        func(ctx context.Context, q DBTX, tenantID int, id int) error
	getByWhatsappIDFunc func(ctx context.Context, q DBTX, whatsappID string) (*WhatsAppDevice, error)
}

func (m *mockWhatsAppDeviceRepository) List(ctx context.Context, q DBTX, tenantID int) ([]WhatsAppDevice, error) {
	return m.listFunc(ctx, q, tenantID)
}

func (m *mockWhatsAppDeviceRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error) {
	return m.getByIDFunc(ctx, q, tenantID, id)
}

func (m *mockWhatsAppDeviceRepository) Create(ctx context.Context, q DBTX, device *WhatsAppDevice) (int, error) {
	return m.createFunc(ctx, q, device)
}

func (m *mockWhatsAppDeviceRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, device *WhatsAppDevice) error {
	return m.updateFunc(ctx, q, tenantID, id, device)
}

func (m *mockWhatsAppDeviceRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	return m.deleteFunc(ctx, q, tenantID, id)
}

func (m *mockWhatsAppDeviceRepository) GetByWhatsappID(ctx context.Context, q DBTX, whatsappID string) (*WhatsAppDevice, error) {
	return m.getByWhatsappIDFunc(ctx, q, whatsappID)
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

func newService(repo WhatsAppDeviceRepository, db *sql.DB) *whatsAppDeviceService {
	return NewWhatsAppDeviceService(repo, db, &config.Config{}, logger.NewLogger("error", "json")).(*whatsAppDeviceService)
}

func TestWhatsAppDeviceService_List(t *testing.T) {
	t.Run("pagination defaults", func(t *testing.T) {
		devices := []WhatsAppDevice{
			{ID: 1, Name: "Device 1", Status: "CONNECTED"},
			{ID: 2, Name: "Device 2", Status: "DISCONNECTED"},
		}
		var capturedTenantID int
		repo := &mockWhatsAppDeviceRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int) ([]WhatsAppDevice, error) {
				capturedTenantID = tenantID
				return devices, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		result, err := svc.List(context.Background(), 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 devices, got %d", len(result))
		}
		if result[0].Name != "Device 1" {
			t.Errorf("expected Name 'Device 1', got %q", result[0].Name)
		}
	})
}

func TestWhatsAppDeviceService_Create(t *testing.T) {
	t.Run("creates device with default status", func(t *testing.T) {
		var captured *WhatsAppDevice
		repo := &mockWhatsAppDeviceRepository{
			createFunc: func(ctx context.Context, q DBTX, device *WhatsAppDevice) (int, error) {
				captured = device
				return 1, nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		req := CreateDeviceRequest{
			Name:        "Test Device",
			CountryCode: "62",
			Phone:       "81234567890",
		}

		result, err := svc.Create(context.Background(), req, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured == nil {
			t.Fatal("expected captured device to be non-nil")
		}
		if captured.Name != "Test Device" {
			t.Errorf("expected Name 'Test Device', got %q", captured.Name)
		}
		if captured.CountryCode != "62" {
			t.Errorf("expected CountryCode '62', got %q", captured.CountryCode)
		}
		if captured.Phone != "81234567890" {
			t.Errorf("expected Phone '81234567890', got %q", captured.Phone)
		}
		expectedID := "test-device-81234567890"
		if captured.WhatsappID != expectedID {
			t.Errorf("expected WhatsappID %q, got %q", expectedID, captured.WhatsappID)
		}
		if captured.Status != "DISCONNECTED" {
			t.Errorf("expected Status 'DISCONNECTED', got %q", captured.Status)
		}
		if captured.TenantID != 1 {
			t.Errorf("expected TenantID 1, got %d", captured.TenantID)
		}
		if result.ID != 1 {
			t.Errorf("expected ID 1, got %d", result.ID)
		}
		if result.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}
		if result.UpdatedAt.IsZero() {
			t.Error("expected UpdatedAt to be set")
		}
	})
}

func TestWhatsAppDeviceService_Update(t *testing.T) {
	t.Run("partial field updates", func(t *testing.T) {
		existing := &WhatsAppDevice{
			ID: 1, TenantID: 1, Name: "Old Name",
			CountryCode: "62", Phone: "81234567890",
			WhatsappID: "whatsapp:old", Status: "CONNECTED",
		}
		var updated *WhatsAppDevice
		repo := &mockWhatsAppDeviceRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error) {
				return existing, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID int, id int, device *WhatsAppDevice) error {
				updated = device
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		newName := "New Device Name"
		req := UpdateDeviceRequest{Name: &newName}

		result, err := svc.Update(context.Background(), 1, 1, req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Name != "New Device Name" {
			t.Errorf("expected Name 'New Device Name', got %q", result.Name)
		}
		if updated == nil {
			t.Fatal("expected updated device to be non-nil")
		}
		if updated.Name != "New Device Name" {
			t.Errorf("expected captured Name 'New Device Name', got %q", updated.Name)
		}
		if updated.CountryCode != "62" {
			t.Errorf("expected CountryCode unchanged '62', got %q", updated.CountryCode)
		}
	})
}

func TestWhatsAppDeviceService_Delete(t *testing.T) {
	t.Run("deletes device", func(t *testing.T) {
		var capturedTenantID, capturedID int
		repo := &mockWhatsAppDeviceRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error) {
				return &WhatsAppDevice{ID: id, TenantID: tenantID, WhatsappID: "test-instance"}, nil
			},
			deleteFunc: func(ctx context.Context, q DBTX, tenantID int, id int) error {
				capturedTenantID = tenantID
				capturedID = id
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		if err := svc.Delete(context.Background(), 1, 5); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if capturedID != 5 {
			t.Errorf("expected id 5, got %d", capturedID)
		}
	})
}

func TestWhatsAppDeviceService_Connect(t *testing.T) {
	t.Run("updates status to connected", func(t *testing.T) {
		existing := &WhatsAppDevice{
			ID: 1, TenantID: 1, Name: "Device",
			WhatsappID: "device-test",
			Status: "DISCONNECTED",
		}
		var updated *WhatsAppDevice
		repo := &mockWhatsAppDeviceRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error) {
				return existing, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID int, id int, device *WhatsAppDevice) error {
				updated = device
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		result, err := svc.Connect(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "CONNECTED" {
			t.Errorf("expected Status 'CONNECTED', got %q", result.Status)
		}
		if updated != nil && updated.Status != "CONNECTED" {
			t.Errorf("expected captured Status 'CONNECTED', got %q", updated.Status)
		}
	})
}

func TestWhatsAppDeviceService_Disconnect(t *testing.T) {
	t.Run("updates status to disconnected", func(t *testing.T) {
		existing := &WhatsAppDevice{
			ID: 1, TenantID: 1, Name: "Device",
			Status: "CONNECTED",
		}
		var updated *WhatsAppDevice
		repo := &mockWhatsAppDeviceRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error) {
				return existing, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID int, id int, device *WhatsAppDevice) error {
				updated = device
				return nil
			},
		}
		m := &mockConn{}
		db := newMockDB(m)
		defer db.Close()
		svc := newService(repo, db)

		result, err := svc.Disconnect(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Status != "DISCONNECTED" {
			t.Errorf("expected Status 'DISCONNECTED', got %q", result.Status)
		}
		if updated != nil && updated.Status != "DISCONNECTED" {
			t.Errorf("expected captured Status 'DISCONNECTED', got %q", updated.Status)
		}
	})
}
