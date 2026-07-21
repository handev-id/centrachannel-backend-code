package campaign

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"io"
	"testing"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/channel"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/logger"
)

// ---- mock driver infrastructure ----

type mockDriver struct{}

func (d *mockDriver) Open(name string) (driver.Conn, error) {
	return &mockConn{}, nil
}

type mockConnector struct {
	conn *mockConn
}

func (c *mockConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return c.conn, nil
}

func (c *mockConnector) Driver() driver.Driver {
	return &mockDriver{}
}

type mockConn struct {
	onBegin func() (driver.Tx, error)
}

func (c *mockConn) Prepare(query string) (driver.Stmt, error) {
	return &mockStmt{}, nil
}

func (c *mockConn) Close() error {
	return nil
}

func (c *mockConn) Begin() (driver.Tx, error) {
	if c.onBegin != nil {
		return c.onBegin()
	}
	return &mockTx{}, nil
}

type mockTx struct {
	commitCalled   bool
	rollbackCalled bool
	commitErr      error
	rollbackErr    error
}

func (t *mockTx) Commit() error {
	t.commitCalled = true
	return t.commitErr
}

func (t *mockTx) Rollback() error {
	t.rollbackCalled = true
	return t.rollbackErr
}

type mockStmt struct{}

func (s *mockStmt) Close() error          { return nil }
func (s *mockStmt) NumInput() int         { return -1 }
func (s *mockStmt) Exec(args []driver.Value) (driver.Result, error) {
	return &mockResult{}, nil
}
func (s *mockStmt) Query(args []driver.Value) (driver.Rows, error) {
	return &mockRows{}, nil
}

type mockResult struct{}

func (r *mockResult) LastInsertId() (int64, error) { return 1, nil }
func (r *mockResult) RowsAffected() (int64, error) { return 1, nil }

type mockRows struct {
	data [][]driver.Value
	pos  int
}

func (r *mockRows) Columns() []string { return nil }
func (r *mockRows) Close() error      { return nil }
func (r *mockRows) Next(dest []driver.Value) error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	copy(dest, r.data[r.pos])
	r.pos++
	return nil
}

// ---- mock repository ----

type mockChannelRepository struct {
	getByIDFunc  func(ctx context.Context, q channel.DBTX, id int) (*channel.Channel, error)
	getByTypeFunc func(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error)
}

func (m *mockChannelRepository) GetByID(ctx context.Context, q channel.DBTX, id int) (*channel.Channel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, id)
	}
	return &channel.Channel{}, nil
}

func (m *mockChannelRepository) GetByType(ctx context.Context, q channel.DBTX, channelType string) (*channel.Channel, error) {
	if m.getByTypeFunc != nil {
		return m.getByTypeFunc(ctx, q, channelType)
	}
	return &channel.Channel{}, nil
}

func (m *mockChannelRepository) List(ctx context.Context, q channel.DBTX) ([]channel.Channel, error) {
	return nil, nil
}

type mockCampaignRepository struct {
	listFunc                   func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]*Campaign, int, error)
	getByIDFunc                func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error)
	createFunc                 func(ctx context.Context, q DBTX, campaign *Campaign) (int, error)
	updateFunc                 func(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error
	deleteFunc                 func(ctx context.Context, q DBTX, tenantID int, id int) error
	createRecipientFunc        func(ctx context.Context, q DBTX, recipient *CampaignRecipient) (int, error)
	updateRecipientStatusFunc  func(ctx context.Context, q DBTX, id int, status string, failedReason *string, deliveryTime *sql.NullTime) error
	countPendingRecipientsFunc func(ctx context.Context, q DBTX, campaignID int) (int, error)
	getPendingRecipientsFunc          func(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignRecipient, error)
	getPendingRecipientsWithPhoneFunc func(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignRecipientWithPhone, error)
	listTemplatesFunc          func(ctx context.Context, q DBTX, tenantID int) ([]CampaignTemplate, error)
	getTemplateByIDFunc        func(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignTemplate, error)
	createTemplateFunc         func(ctx context.Context, q DBTX, template *CampaignTemplate) (int, error)
	updateTemplateFunc         func(ctx context.Context, q DBTX, tenantID int, id int, template *CampaignTemplate) error
	deleteTemplateFunc         func(ctx context.Context, q DBTX, tenantID int, id int) error
	listRecipientListsFunc     func(ctx context.Context, q DBTX, tenantID int) ([]CampaignRecipientList, error)
	getRecipientListByIDFunc   func(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignRecipientList, error)
	createRecipientListFunc    func(ctx context.Context, q DBTX, list *CampaignRecipientList) (int, error)
	updateRecipientListFunc    func(ctx context.Context, q DBTX, tenantID int, id int, list *CampaignRecipientList) error
	deleteRecipientListFunc    func(ctx context.Context, q DBTX, tenantID int, id int) error
	listRecipientContactsFunc  func(ctx context.Context, q DBTX, listID int) ([]CampaignRecipientContact, error)
	createRecipientContactFunc func(ctx context.Context, q DBTX, contact *CampaignRecipientContact) (int, error)
	deleteRecipientContactFunc func(ctx context.Context, q DBTX, id int) error
}

func (m *mockCampaignRepository) List(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]*Campaign, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, q, tenantID, limit, offset, search)
	}
	return nil, 0, nil
}

func (m *mockCampaignRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, q, tenantID, id)
	}
	return &Campaign{Status: "draft"}, nil
}

func (m *mockCampaignRepository) Create(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
	if m.createFunc != nil {
		return m.createFunc(ctx, q, campaign)
	}
	return 1, nil
}

func (m *mockCampaignRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, q, tenantID, id, campaign)
	}
	return nil
}

func (m *mockCampaignRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, q, tenantID, id)
	}
	return nil
}

func (m *mockCampaignRepository) CreateRecipient(ctx context.Context, q DBTX, recipient *CampaignRecipient) (int, error) {
	if m.createRecipientFunc != nil {
		return m.createRecipientFunc(ctx, q, recipient)
	}
	return 1, nil
}

func (m *mockCampaignRepository) UpdateRecipientStatus(ctx context.Context, q DBTX, id int, status string, failedReason *string, deliveryTime *sql.NullTime) error {
	if m.updateRecipientStatusFunc != nil {
		return m.updateRecipientStatusFunc(ctx, q, id, status, failedReason, deliveryTime)
	}
	return nil
}

func (m *mockCampaignRepository) CountPendingRecipients(ctx context.Context, q DBTX, campaignID int) (int, error) {
	if m.countPendingRecipientsFunc != nil {
		return m.countPendingRecipientsFunc(ctx, q, campaignID)
	}
	return 0, nil
}

func (m *mockCampaignRepository) GetPendingRecipients(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignRecipient, error) {
	if m.getPendingRecipientsFunc != nil {
		return m.getPendingRecipientsFunc(ctx, q, campaignID, limit)
	}
	return nil, nil
}

func (m *mockCampaignRepository) GetPendingRecipientsWithPhone(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignRecipientWithPhone, error) {
	if m.getPendingRecipientsWithPhoneFunc != nil {
		return m.getPendingRecipientsWithPhoneFunc(ctx, q, campaignID, limit)
	}
	return nil, nil
}

func (m *mockCampaignRepository) ListTemplates(ctx context.Context, q DBTX, tenantID int) ([]CampaignTemplate, error) {
	if m.listTemplatesFunc != nil {
		return m.listTemplatesFunc(ctx, q, tenantID)
	}
	return nil, nil
}

func (m *mockCampaignRepository) GetTemplateByID(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignTemplate, error) {
	if m.getTemplateByIDFunc != nil {
		return m.getTemplateByIDFunc(ctx, q, tenantID, id)
	}
	return nil, nil
}

func (m *mockCampaignRepository) CreateTemplate(ctx context.Context, q DBTX, template *CampaignTemplate) (int, error) {
	if m.createTemplateFunc != nil {
		return m.createTemplateFunc(ctx, q, template)
	}
	return 1, nil
}

func (m *mockCampaignRepository) UpdateTemplate(ctx context.Context, q DBTX, tenantID int, id int, template *CampaignTemplate) error {
	if m.updateTemplateFunc != nil {
		return m.updateTemplateFunc(ctx, q, tenantID, id, template)
	}
	return nil
}

func (m *mockCampaignRepository) DeleteTemplate(ctx context.Context, q DBTX, tenantID int, id int) error {
	if m.deleteTemplateFunc != nil {
		return m.deleteTemplateFunc(ctx, q, tenantID, id)
	}
	return nil
}

func (m *mockCampaignRepository) ListRecipientLists(ctx context.Context, q DBTX, tenantID int) ([]CampaignRecipientList, error) {
	if m.listRecipientListsFunc != nil {
		return m.listRecipientListsFunc(ctx, q, tenantID)
	}
	return nil, nil
}

func (m *mockCampaignRepository) GetRecipientListByID(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignRecipientList, error) {
	if m.getRecipientListByIDFunc != nil {
		return m.getRecipientListByIDFunc(ctx, q, tenantID, id)
	}
	return nil, nil
}

func (m *mockCampaignRepository) CreateRecipientList(ctx context.Context, q DBTX, list *CampaignRecipientList) (int, error) {
	if m.createRecipientListFunc != nil {
		return m.createRecipientListFunc(ctx, q, list)
	}
	return 1, nil
}

func (m *mockCampaignRepository) UpdateRecipientList(ctx context.Context, q DBTX, tenantID int, id int, list *CampaignRecipientList) error {
	if m.updateRecipientListFunc != nil {
		return m.updateRecipientListFunc(ctx, q, tenantID, id, list)
	}
	return nil
}

func (m *mockCampaignRepository) DeleteRecipientList(ctx context.Context, q DBTX, tenantID int, id int) error {
	if m.deleteRecipientListFunc != nil {
		return m.deleteRecipientListFunc(ctx, q, tenantID, id)
	}
	return nil
}

func (m *mockCampaignRepository) ListRecipientContacts(ctx context.Context, q DBTX, listID int) ([]CampaignRecipientContact, error) {
	if m.listRecipientContactsFunc != nil {
		return m.listRecipientContactsFunc(ctx, q, listID)
	}
	return nil, nil
}

func (m *mockCampaignRepository) CreateRecipientContact(ctx context.Context, q DBTX, contact *CampaignRecipientContact) (int, error) {
	if m.createRecipientContactFunc != nil {
		return m.createRecipientContactFunc(ctx, q, contact)
	}
	return 1, nil
}

func (m *mockCampaignRepository) DeleteRecipientContact(ctx context.Context, q DBTX, id int) error {
	if m.deleteRecipientContactFunc != nil {
		return m.deleteRecipientContactFunc(ctx, q, id)
	}
	return nil
}

// ---- helpers ----

func newMockDB(beginFn func() (driver.Tx, error)) *sql.DB {
	return sql.OpenDB(&mockConnector{
		conn: &mockConn{onBegin: beginFn},
	})
}

func testConfig() *config.Config {
	return &config.Config{}
}

func testLogger() *logger.Logger {
	return logger.NewLogger("error", "text")
}

func testTenant() *tenant.Tenant {
	return &tenant.Tenant{ID: 1, Name: "Test Tenant"}
}

// ---- tests ----

func TestCampaignService(t *testing.T) {
	t.Run("List defaults pagination", func(t *testing.T) {
		var capturedLimit, capturedOffset, capturedTenantID int
		repo := &mockCampaignRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]*Campaign, int, error) {
				capturedLimit = limit
				capturedOffset = offset
				capturedTenantID = tenantID
				return []*Campaign{{ID: 1, Name: "Campaign 1"}}, 1, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, total, err := svc.List(context.Background(), ListCampaignQuery{}, testTenant())
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if capturedLimit != 20 {
			t.Errorf("expected default limit 20, got %d", capturedLimit)
		}
		if capturedOffset != 0 {
			t.Errorf("expected default offset 0, got %d", capturedOffset)
		}
		if capturedTenantID != 1 {
			t.Errorf("expected tenantID 1, got %d", capturedTenantID)
		}
		if total != 1 {
			t.Errorf("expected total 1, got %d", total)
		}
		if len(result) != 1 {
			t.Errorf("expected 1 campaign, got %d", len(result))
		}
	})

	t.Run("List custom pagination", func(t *testing.T) {
		var capturedLimit, capturedOffset int
		repo := &mockCampaignRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]*Campaign, int, error) {
				capturedLimit = limit
				capturedOffset = offset
				return []*Campaign{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}, 10, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, total, err := svc.List(context.Background(), ListCampaignQuery{Page: 2, Limit: 5}, testTenant())
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if capturedLimit != 5 {
			t.Errorf("expected limit 5, got %d", capturedLimit)
		}
		if capturedOffset != 5 {
			t.Errorf("expected offset 5, got %d", capturedOffset)
		}
		if total != 10 {
			t.Errorf("expected total 10, got %d", total)
		}
		if len(result) != 2 {
			t.Errorf("expected 2 campaigns, got %d", len(result))
		}
	})

	t.Run("List clamps limit to 100", func(t *testing.T) {
		var capturedLimit int
		repo := &mockCampaignRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]*Campaign, int, error) {
				capturedLimit = limit
				return nil, 0, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		_, _, err := svc.List(context.Background(), ListCampaignQuery{Page: 1, Limit: 200}, testTenant())
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if capturedLimit != 20 {
			t.Errorf("expected limit clamped to 20, got %d", capturedLimit)
		}
	})

	t.Run("Create campaign with recipient list copies contacts", func(t *testing.T) {
		createdRecipients := make([]*CampaignRecipient, 0)
		mockTxObj := &mockTx{}

		db := newMockDB(func() (driver.Tx, error) {
			return mockTxObj, nil
		})
		defer db.Close()

		recipientListID := 10
		repo := &mockCampaignRepository{
			createFunc: func(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
				if campaign.RecipientListID == nil || *campaign.RecipientListID != recipientListID {
					t.Errorf("expected RecipientListID %d", recipientListID)
				}
				if campaign.TotalCount != 0 {
					t.Errorf("expected TotalCount 0 when passed to Create, got %d", campaign.TotalCount)
				}
				return 7, nil
			},
			listRecipientContactsFunc: func(ctx context.Context, q DBTX, listID int) ([]CampaignRecipientContact, error) {
				if listID != recipientListID {
					t.Errorf("expected listID %d, got %d", recipientListID, listID)
				}
				return []CampaignRecipientContact{
					{ID: 101, FirstName: "Alice"},
					{ID: 102, FirstName: "Bob"},
				}, nil
			},
			createRecipientFunc: func(ctx context.Context, q DBTX, recipient *CampaignRecipient) (int, error) {
				createdRecipients = append(createdRecipients, recipient)
				return 1, nil
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

		result, err := svc.Create(context.Background(), CreateCampaignRequest{
			Name:            "Test Campaign",
			MessageTemplate: "Hello",
			ChannelID:       5,
			RecipientListID: &recipientListID,
		}, testTenant(), 42)
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		if result.ID != 7 {
			t.Errorf("expected campaign ID 7, got %d", result.ID)
		}
		if result.TotalCount != 2 {
			t.Errorf("expected TotalCount 2, got %d", result.TotalCount)
		}
		if !mockTxObj.commitCalled {
			t.Error("expected transaction to be committed")
		}
		if mockTxObj.rollbackCalled {
			t.Error("expected rollback not to be called when commit succeeds")
		}
		if len(createdRecipients) != 2 {
			t.Fatalf("expected 2 recipients, got %d", len(createdRecipients))
		}
		if createdRecipients[0].CampaignID != 7 {
			t.Errorf("expected recipient CampaignID 7, got %d", createdRecipients[0].CampaignID)
		}
		if createdRecipients[0].RecipientContactID != 101 {
			t.Errorf("expected RecipientContactID 101, got %d", createdRecipients[0].RecipientContactID)
		}
		if createdRecipients[0].Status != "pending" {
			t.Errorf("expected Status 'pending', got %s", createdRecipients[0].Status)
		}
		if createdRecipients[1].RecipientContactID != 102 {
			t.Errorf("expected RecipientContactID 102, got %d", createdRecipients[1].RecipientContactID)
		}
	})

	t.Run("Create with scheduled_at", func(t *testing.T) {
		mockTxObj := &mockTx{}
		db := newMockDB(func() (driver.Tx, error) {
			return mockTxObj, nil
		})
		defer db.Close()

		var capturedCampaign *Campaign
		repo := &mockCampaignRepository{
			createFunc: func(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
				capturedCampaign = campaign
				return 1, nil
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

		scheduledStr := "2026-06-15T10:00:00Z"
		_, err := svc.Create(context.Background(), CreateCampaignRequest{
			Name:            "Scheduled",
			MessageTemplate: "Hi",
			ChannelID:       1,
			ScheduledAt:     &scheduledStr,
		}, testTenant(), 1)
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		if capturedCampaign.ScheduledAt == nil {
			t.Fatal("expected ScheduledAt to be set")
		}
		expected, _ := time.Parse(time.RFC3339, scheduledStr)
		if !capturedCampaign.ScheduledAt.Equal(expected) {
			t.Errorf("expected ScheduledAt %v, got %v", expected, *capturedCampaign.ScheduledAt)
		}
	})

	t.Run("Create invalid scheduled_at format", func(t *testing.T) {
		db := newMockDB(func() (driver.Tx, error) {
			return &mockTx{}, nil
		})
		defer db.Close()

		repo := &mockCampaignRepository{
			createFunc: func(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
				return 1, nil
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

		badTime := "not-a-time"
		_, err := svc.Create(context.Background(), CreateCampaignRequest{
			Name:            "Bad",
			MessageTemplate: "Hi",
			ChannelID:       1,
			ScheduledAt:     &badTime,
		}, testTenant(), 1)
		if err == nil {
			t.Fatal("expected error for invalid scheduled_at format")
		}
	})

	t.Run("Create with recipient list and transaction rollback on error", func(t *testing.T) {
		mockTxObj := &mockTx{}
		db := newMockDB(func() (driver.Tx, error) {
			return mockTxObj, nil
		})
		defer db.Close()

		recipientListID := 10
		repo := &mockCampaignRepository{
			createFunc: func(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
				return 7, nil
			},
			listRecipientContactsFunc: func(ctx context.Context, q DBTX, listID int) ([]CampaignRecipientContact, error) {
				return []CampaignRecipientContact{
					{ID: 101, FirstName: "Alice"},
				}, nil
			},
			createRecipientFunc: func(ctx context.Context, q DBTX, recipient *CampaignRecipient) (int, error) {
				return 0, sql.ErrConnDone
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

		_, err := svc.Create(context.Background(), CreateCampaignRequest{
			Name:            "Fail",
			MessageTemplate: "Hi",
			ChannelID:       1,
			RecipientListID: &recipientListID,
		}, testTenant(), 1)
		if err == nil {
			t.Fatal("expected error when CreateRecipient fails")
		}
		if mockTxObj.commitCalled {
			t.Error("expected transaction NOT to be committed on error")
		}
	})

	t.Run("Update partial fields", func(t *testing.T) {
		var capturedUpdate *Campaign
		getByIDCalls := 0

		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				getByIDCalls++
				if getByIDCalls == 1 {
					return &Campaign{
						ID:              1,
						Name:            "Original Name",
						Description:     nil,
						MessageTemplate: "Original Template",
						Status:          "draft",
					}, nil
				}
				return &Campaign{
					ID:              1,
					Name:            "Updated Name",
					Description:     nil,
					MessageTemplate: "Original Template",
					Status:          "archived",
				}, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error {
				capturedUpdate = campaign
				return nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		newName := "Updated Name"
		newStatus := "archived"
		result, err := svc.Update(context.Background(), 1, 1, UpdateCampaignRequest{
			Name:   &newName,
			Status: &newStatus,
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		if getByIDCalls != 2 {
			t.Errorf("expected 2 GetByID calls (before update + return), got %d", getByIDCalls)
		}
		if capturedUpdate.Name != "Updated Name" {
			t.Errorf("expected Name 'Updated Name', got %s", capturedUpdate.Name)
		}
		if capturedUpdate.Status != "archived" {
			t.Errorf("expected Status 'archived', got %s", capturedUpdate.Status)
		}
		if capturedUpdate.MessageTemplate != "Original Template" {
			t.Errorf("expected MessageTemplate to remain 'Original Template', got %s", capturedUpdate.MessageTemplate)
		}
		if capturedUpdate.Description != nil {
			t.Errorf("expected Description to remain nil, got %v", *capturedUpdate.Description)
		}
		if result.Name != "Updated Name" {
			t.Errorf("expected result Name 'Updated Name', got %s", result.Name)
		}
	})

	t.Run("Update only name", func(t *testing.T) {
		var capturedUpdate *Campaign
		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return &Campaign{
					ID:              1,
					Name:            "Old",
					MessageTemplate: "Tmpl",
					Status:          "draft",
				}, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error {
				capturedUpdate = campaign
				return nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		newName := "New Name"
		_, err := svc.Update(context.Background(), 1, 1, UpdateCampaignRequest{
			Name: &newName,
		})
		if err != nil {
			t.Fatalf("Update returned error: %v", err)
		}
		if capturedUpdate.Name != "New Name" {
			t.Errorf("expected Name 'New Name', got %s", capturedUpdate.Name)
		}
		if capturedUpdate.Status != "draft" {
			t.Errorf("expected Status to remain 'draft', got %s", capturedUpdate.Status)
		}
		if capturedUpdate.MessageTemplate != "Tmpl" {
			t.Errorf("expected MessageTemplate to remain 'Tmpl', got %s", capturedUpdate.MessageTemplate)
		}
	})

	t.Run("Update non-draft returns error", func(t *testing.T) {
		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return &Campaign{
					ID:     1,
					Name:   "Sent Campaign",
					Status: "sent",
				}, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		newName := "Should Fail"
		_, err := svc.Update(context.Background(), 1, 1, UpdateCampaignRequest{
			Name: &newName,
		})
		if err == nil {
			t.Fatal("expected error when updating non-draft campaign")
		}
	})

	t.Run("Update campaign not found", func(t *testing.T) {
		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return nil, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		newName := "Nope"
		_, err := svc.Update(context.Background(), 1, 999, UpdateCampaignRequest{Name: &newName})
		if err == nil {
			t.Fatal("expected error for non-existent campaign")
		}
	})

	t.Run("Update invalid scheduled_at", func(t *testing.T) {
		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return &Campaign{ID: 1, Name: "A", Status: "draft"}, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		bad := "bad-date"
		_, err := svc.Update(context.Background(), 1, 1, UpdateCampaignRequest{ScheduledAt: &bad})
		if err == nil {
			t.Fatal("expected error for invalid scheduled_at format")
		}
	})

	t.Run("Delete delegates to repository", func(t *testing.T) {
		deleteCalled := false
		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return &Campaign{ID: 1, Name: "Draft", Status: "draft"}, nil
			},
			deleteFunc: func(ctx context.Context, q DBTX, tenantID int, id int) error {
				deleteCalled = true
				if tenantID != 1 {
					t.Errorf("expected tenantID 1, got %d", tenantID)
				}
				if id != 5 {
					t.Errorf("expected id 5, got %d", id)
				}
				return nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		err := svc.Delete(context.Background(), 1, 5)
		if err != nil {
			t.Fatalf("Delete returned error: %v", err)
		}
		if !deleteCalled {
			t.Error("expected repo.Delete to be called")
		}
	})

	t.Run("Delete protected statuses", func(t *testing.T) {
		tests := []struct {
			name   string
			status string
		}{
			{"sending status", "sending"},
			{"sent status", "sent"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				deleteCalled := false
				repo := &mockCampaignRepository{
					getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
						return &Campaign{ID: 1, Status: tt.status}, nil
					},
					deleteFunc: func(ctx context.Context, q DBTX, tenantID int, id int) error {
						deleteCalled = true
						return nil
					},
				}
				svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

				err := svc.Delete(context.Background(), 1, 1)
				if err == nil {
					t.Errorf("expected error for %s campaign", tt.status)
				}
				if deleteCalled {
					t.Error("repo.Delete should not be called")
				}
			})
		}
	})

	t.Run("Delete not found", func(t *testing.T) {
		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return nil, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		err := svc.Delete(context.Background(), 1, 999)
		if err == nil {
			t.Fatal("expected error for non-existent campaign")
		}
	})

	t.Run("Send updates status to sending then processes recipients", func(t *testing.T) {
		var updateStatusCalls []string
		mockTxObj := &mockTx{}

		db := newMockDB(func() (driver.Tx, error) {
			return mockTxObj, nil
		})
		defer db.Close()

		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return &Campaign{ID: 1, Name: "Test", Status: "draft"}, nil
			},
			updateFunc: func(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error {
				updateStatusCalls = append(updateStatusCalls, campaign.Status)
				return nil
			},
			getPendingRecipientsFunc: func(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignRecipient, error) {
				return nil, nil
			},
			countPendingRecipientsFunc: func(ctx context.Context, q DBTX, campaignID int) (int, error) {
				return 0, nil
			},
		}
		mockChannelRepo := &mockChannelRepository{
			getByIDFunc: func(ctx context.Context, q channel.DBTX, id int) (*channel.Channel, error) {
				return &channel.Channel{ID: 1, Type: "whatsapp"}, nil
			},
		}
		svc := &campaignService{repo: repo, channelRepo: mockChannelRepo, db: db, cfg: testConfig(), logger: testLogger()}

		err := svc.Send(context.Background(), 1, 1)
		if err != nil {
			t.Fatalf("Send returned error: %v", err)
		}
		if !mockTxObj.commitCalled {
			t.Error("expected transaction to be committed")
		}
		if len(updateStatusCalls) < 1 {
			t.Fatal("expected at least 1 Update call")
		}
		if updateStatusCalls[0] != "sending" {
			t.Errorf("expected status update to 'sending', got %s", updateStatusCalls[0])
		}
	})

	t.Run("Send with non-draft and non-failed status returns error", func(t *testing.T) {
		tests := []struct {
			name   string
			status string
		}{
			{"sending status", "sending"},
			{"sent status", "sent"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				repo := &mockCampaignRepository{
					getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
						return &Campaign{ID: 1, Status: tt.status}, nil
					},
				}
				svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger(), deviceRepo: nil, channelRepo: nil}

				err := svc.Send(context.Background(), 1, 1)
				if err == nil {
					t.Errorf("expected error for %s campaign", tt.status)
				}
			})
		}
	})

	t.Run("Send campaign not found", func(t *testing.T) {
		repo := &mockCampaignRepository{
			getByIDFunc: func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
				return nil, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger(), deviceRepo: nil, channelRepo: nil}

		err := svc.Send(context.Background(), 1, 999)
		if err == nil {
			t.Fatal("expected error for non-existent campaign")
		}
	})

	t.Run("ListTemplates returns templates", func(t *testing.T) {
		repo := &mockCampaignRepository{
			listTemplatesFunc: func(ctx context.Context, q DBTX, tenantID int) ([]CampaignTemplate, error) {
				if tenantID != 1 {
					t.Errorf("expected tenantID 1, got %d", tenantID)
				}
				return []CampaignTemplate{
					{ID: 1, Name: "Welcome", Content: json.RawMessage(`{"text":"Hello"}`)},
					{ID: 2, Name: "Promo", Content: json.RawMessage(`{"text":"Sale"}`)},
				}, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		templates, err := svc.ListTemplates(context.Background(), 1)
		if err != nil {
			t.Fatalf("ListTemplates returned error: %v", err)
		}
		if len(templates) != 2 {
			t.Fatalf("expected 2 templates, got %d", len(templates))
		}
		if templates[0].Name != "Welcome" {
			t.Errorf("expected Name 'Welcome', got %s", templates[0].Name)
		}
		if templates[1].Name != "Promo" {
			t.Errorf("expected Name 'Promo', got %s", templates[1].Name)
		}
	})

	t.Run("CreateTemplate creates with content and variables", func(t *testing.T) {
		var capturedTemplate *CampaignTemplate
		repo := &mockCampaignRepository{
			createTemplateFunc: func(ctx context.Context, q DBTX, template *CampaignTemplate) (int, error) {
				capturedTemplate = template
				return 3, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		content := json.RawMessage(`{"text":"Hello {{name}}"}`)
		variables := json.RawMessage(`[{"name":"name","type":"text"}]`)
		result, err := svc.CreateTemplate(context.Background(), CreateTemplateRequest{
			Name:         "Test Template",
			Content:      content,
			Variables:    variables,
			TemplateType: strPtr("text"),
			Language:     strPtr("en"),
			Category:     strPtr("utility"),
		}, testTenant())
		if err != nil {
			t.Fatalf("CreateTemplate returned error: %v", err)
		}
		if capturedTemplate.TenantID != 1 {
			t.Errorf("expected TenantID 1, got %d", capturedTemplate.TenantID)
		}
		if capturedTemplate.Name != "Test Template" {
			t.Errorf("expected Name 'Test Template', got %s", capturedTemplate.Name)
		}
		if string(capturedTemplate.Content) != string(content) {
			t.Errorf("expected Content %s, got %s", content, capturedTemplate.Content)
		}
		if string(capturedTemplate.Variables) != string(variables) {
			t.Errorf("expected Variables %s, got %s", variables, capturedTemplate.Variables)
		}
		if result.ID != 3 {
			t.Errorf("expected template ID 3, got %d", result.ID)
		}
	})

	t.Run("CreateTemplate defaults variables to empty array", func(t *testing.T) {
		var capturedTemplate *CampaignTemplate
		repo := &mockCampaignRepository{
			createTemplateFunc: func(ctx context.Context, q DBTX, template *CampaignTemplate) (int, error) {
				capturedTemplate = template
				return 1, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		content := json.RawMessage(`{"text":"Hi"}`)
		_, err := svc.CreateTemplate(context.Background(), CreateTemplateRequest{
			Name:    "Simple",
			Content: content,
		}, testTenant())
		if err != nil {
			t.Fatalf("CreateTemplate returned error: %v", err)
		}
		if string(capturedTemplate.Variables) != "[]" {
			t.Errorf("expected Variables default '[]', got %s", string(capturedTemplate.Variables))
		}
	})

	t.Run("ListRecipientLists returns lists", func(t *testing.T) {
		repo := &mockCampaignRepository{
			listRecipientListsFunc: func(ctx context.Context, q DBTX, tenantID int) ([]CampaignRecipientList, error) {
				if tenantID != 1 {
					t.Errorf("expected tenantID 1, got %d", tenantID)
				}
				return []CampaignRecipientList{
					{ID: 1, Name: "Newsletter", Source: "manual"},
					{ID: 2, Name: "VIP Customers", Source: "import"},
				}, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		lists, err := svc.ListRecipientLists(context.Background(), 1)
		if err != nil {
			t.Fatalf("ListRecipientLists returned error: %v", err)
		}
		if len(lists) != 2 {
			t.Fatalf("expected 2 lists, got %d", len(lists))
		}
		if lists[0].Name != "Newsletter" {
			t.Errorf("expected Name 'Newsletter', got %s", lists[0].Name)
		}
		if lists[1].Name != "VIP Customers" {
			t.Errorf("expected Name 'VIP Customers', got %s", lists[1].Name)
		}
	})

	t.Run("CreateRecipientList creates with source defaulting to manual", func(t *testing.T) {
		var capturedList *CampaignRecipientList
		repo := &mockCampaignRepository{
			createRecipientListFunc: func(ctx context.Context, q DBTX, list *CampaignRecipientList) (int, error) {
				capturedList = list
				return 5, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, err := svc.CreateRecipientList(context.Background(), CreateRecipientListRequest{
			Name: "My List",
		}, testTenant())
		if err != nil {
			t.Fatalf("CreateRecipientList returned error: %v", err)
		}
		if capturedList.TenantID != 1 {
			t.Errorf("expected TenantID 1, got %d", capturedList.TenantID)
		}
		if capturedList.Name != "My List" {
			t.Errorf("expected Name 'My List', got %s", capturedList.Name)
		}
		if capturedList.Source != "manual" {
			t.Errorf("expected Source default 'manual', got %s", capturedList.Source)
		}
		if result.ID != 5 {
			t.Errorf("expected list ID 5, got %d", result.ID)
		}
	})

	t.Run("CreateRecipientList preserves provided source", func(t *testing.T) {
		var capturedList *CampaignRecipientList
		repo := &mockCampaignRepository{
			createRecipientListFunc: func(ctx context.Context, q DBTX, list *CampaignRecipientList) (int, error) {
				capturedList = list
				return 6, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		_, err := svc.CreateRecipientList(context.Background(), CreateRecipientListRequest{
			Name:   "Imported",
			Source: "import",
		}, testTenant())
		if err != nil {
			t.Fatalf("CreateRecipientList returned error: %v", err)
		}
		if capturedList.Source != "import" {
			t.Errorf("expected Source 'import', got %s", capturedList.Source)
		}
	})

	t.Run("AddRecipientContact adds contact to list", func(t *testing.T) {
		var capturedContact *CampaignRecipientContact
		repo := &mockCampaignRepository{
			createRecipientContactFunc: func(ctx context.Context, q DBTX, contact *CampaignRecipientContact) (int, error) {
				capturedContact = contact
				return 20, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, err := svc.AddRecipientContact(context.Background(), 5, AddContactToListRequest{
			FirstName: "John",
			LastName:  strPtr("Doe"),
			Email:     strPtr("john@example.com"),
			Phone:     strPtr("+123456789"),
		})
		if err != nil {
			t.Fatalf("AddRecipientContact returned error: %v", err)
		}
		if capturedContact.CampaignRecipientListID != 5 {
			t.Errorf("expected CampaignRecipientListID 5, got %d", capturedContact.CampaignRecipientListID)
		}
		if capturedContact.FirstName != "John" {
			t.Errorf("expected FirstName 'John', got %s", capturedContact.FirstName)
		}
		if capturedContact.LastName == nil || *capturedContact.LastName != "Doe" {
			t.Errorf("expected LastName 'Doe', got %v", *capturedContact.LastName)
		}
		if capturedContact.Email == nil || *capturedContact.Email != "john@example.com" {
			t.Errorf("expected Email 'john@example.com', got %v", *capturedContact.Email)
		}
		if result.ID != 20 {
			t.Errorf("expected contact ID 20, got %d", result.ID)
		}
	})
}

func strPtr(s string) *string {
	return &s
}
