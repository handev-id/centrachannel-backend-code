package campaign

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"io"
	"testing"
	"time"

	"centrachannel/config"
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

type mockCampaignRepository struct {
	listFunc                func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error)
	getByIDFunc             func(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error)
	createFunc              func(ctx context.Context, q DBTX, campaign *Campaign) (int, error)
	updateFunc              func(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error
	deleteFunc              func(ctx context.Context, q DBTX, tenantID int, id int) error
	createContactFunc       func(ctx context.Context, q DBTX, campaignID int, contactID int) error
	updateContactStatusFunc func(ctx context.Context, q DBTX, campaignID int, contactID int, status string, errorMsg *string) error
	countPendingFunc        func(ctx context.Context, q DBTX, campaignID int) (int, error)
	getPendingFunc          func(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignContact, error)
}

func (m *mockCampaignRepository) List(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error) {
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

func (m *mockCampaignRepository) CreateContact(ctx context.Context, q DBTX, campaignID int, contactID int) error {
	if m.createContactFunc != nil {
		return m.createContactFunc(ctx, q, campaignID, contactID)
	}
	return nil
}

func (m *mockCampaignRepository) UpdateContactStatus(ctx context.Context, q DBTX, campaignID int, contactID int, status string, errorMsg *string) error {
	if m.updateContactStatusFunc != nil {
		return m.updateContactStatusFunc(ctx, q, campaignID, contactID, status, errorMsg)
	}
	return nil
}

func (m *mockCampaignRepository) CountPendingContacts(ctx context.Context, q DBTX, campaignID int) (int, error) {
	if m.countPendingFunc != nil {
		return m.countPendingFunc(ctx, q, campaignID)
	}
	return 0, nil
}

func (m *mockCampaignRepository) GetPendingContacts(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignContact, error) {
	if m.getPendingFunc != nil {
		return m.getPendingFunc(ctx, q, campaignID, limit)
	}
	return nil, nil
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
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error) {
				capturedLimit = limit
				capturedOffset = offset
				capturedTenantID = tenantID
				return []Campaign{{ID: 1, Name: "Campaign 1"}}, 1, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, err := svc.List(context.Background(), ListCampaignQuery{}, testTenant())
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
		if result.Meta.CurrentPage != 1 {
			t.Errorf("expected CurrentPage 1, got %d", result.Meta.CurrentPage)
		}
		if result.Meta.PerPage != 20 {
			t.Errorf("expected PerPage 20, got %d", result.Meta.PerPage)
		}
		if result.Meta.Total != 1 {
			t.Errorf("expected Total 1, got %d", result.Meta.Total)
		}
		if result.Meta.LastPage != 1 {
			t.Errorf("expected LastPage 1, got %d", result.Meta.LastPage)
		}
		if result.Meta.From != 1 {
			t.Errorf("expected From 1, got %d", result.Meta.From)
		}
		if result.Meta.To != 1 {
			t.Errorf("expected To 1, got %d", result.Meta.To)
		}
	})

	t.Run("List custom pagination", func(t *testing.T) {
		var capturedLimit, capturedOffset int
		repo := &mockCampaignRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error) {
				capturedLimit = limit
				capturedOffset = offset
				return []Campaign{{ID: 1, Name: "A"}, {ID: 2, Name: "B"}}, 10, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		result, err := svc.List(context.Background(), ListCampaignQuery{Page: 2, Limit: 5}, testTenant())
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if capturedLimit != 5 {
			t.Errorf("expected limit 5, got %d", capturedLimit)
		}
		if capturedOffset != 5 {
			t.Errorf("expected offset 5, got %d", capturedOffset)
		}
		if result.Meta.CurrentPage != 2 {
			t.Errorf("expected CurrentPage 2, got %d", result.Meta.CurrentPage)
		}
		if result.Meta.PerPage != 5 {
			t.Errorf("expected PerPage 5, got %d", result.Meta.PerPage)
		}
		if result.Meta.LastPage != 2 {
			t.Errorf("expected LastPage 2, got %d", result.Meta.LastPage)
		}
		if result.Meta.From != 6 {
			t.Errorf("expected From 6, got %d", result.Meta.From)
		}
		if result.Meta.To != 7 {
			t.Errorf("expected To 7, got %d", result.Meta.To)
		}
	})

	t.Run("List clamps limit to 100", func(t *testing.T) {
		var capturedLimit int
		repo := &mockCampaignRepository{
			listFunc: func(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error) {
				capturedLimit = limit
				return nil, 0, nil
			},
		}
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		_, err := svc.List(context.Background(), ListCampaignQuery{Page: 1, Limit: 200}, testTenant())
		if err != nil {
			t.Fatalf("List returned error: %v", err)
		}
		if capturedLimit != 20 {
			t.Errorf("expected limit clamped to 20, got %d", capturedLimit)
		}
	})

	t.Run("Create campaign with contacts in transaction", func(t *testing.T) {
		createdContactIDs := make([]int, 0)
		var txPassedToCreate DBTX
		mockTxObj := &mockTx{}

		db := newMockDB(func() (driver.Tx, error) {
			return mockTxObj, nil
		})
		defer db.Close()

		repo := &mockCampaignRepository{
			createFunc: func(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
				txPassedToCreate = q
				if campaign.Status != "draft" {
					t.Errorf("expected Status draft, got %s", campaign.Status)
				}
				if campaign.TenantID != 1 {
					t.Errorf("expected TenantID 1, got %d", campaign.TenantID)
				}
				if campaign.Name != "Test Campaign" {
					t.Errorf("expected Name 'Test Campaign', got %s", campaign.Name)
				}
				if campaign.MessageTemplate != "Hello" {
					t.Errorf("expected MessageTemplate 'Hello', got %s", campaign.MessageTemplate)
				}
				if campaign.ChannelID != 5 {
					t.Errorf("expected ChannelID 5, got %d", campaign.ChannelID)
				}
				if campaign.TotalCount != 2 {
					t.Errorf("expected TotalCount 2, got %d", campaign.TotalCount)
				}
				if campaign.CreatedBy != 42 {
					t.Errorf("expected CreatedBy 42, got %d", campaign.CreatedBy)
				}
				return 7, nil
			},
			createContactFunc: func(ctx context.Context, q DBTX, campaignID int, contactID int) error {
				if q != txPassedToCreate {
					t.Error("CreateContact must use the same transaction as Create")
				}
				if campaignID != 7 {
					t.Errorf("expected campaignID 7, got %d", campaignID)
				}
				createdContactIDs = append(createdContactIDs, contactID)
				return nil
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

		result, err := svc.Create(context.Background(), CreateCampaignRequest{
			Name:            "Test Campaign",
			MessageTemplate: "Hello",
			ChannelID:       5,
			ContactIDs:      []int{101, 102},
		}, testTenant(), 42)
		if err != nil {
			t.Fatalf("Create returned error: %v", err)
		}
		if result.ID != 7 {
			t.Errorf("expected campaign ID 7, got %d", result.ID)
		}
		if !mockTxObj.commitCalled {
			t.Error("expected transaction to be committed")
		}
		if mockTxObj.rollbackCalled {
			t.Error("expected rollback not to be called when commit succeeds")
		}
		if len(createdContactIDs) != 2 {
			t.Fatalf("expected 2 contacts, got %d", len(createdContactIDs))
		}
		if createdContactIDs[0] != 101 || createdContactIDs[1] != 102 {
			t.Errorf("expected contact IDs [101 102], got %v", createdContactIDs)
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
			createContactFunc: func(ctx context.Context, q DBTX, campaignID int, contactID int) error {
				return nil
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

		scheduledStr := "2026-06-15T10:00:00Z"
		_, err := svc.Create(context.Background(), CreateCampaignRequest{
			Name:            "Scheduled",
			MessageTemplate: "Hi",
			ChannelID:       1,
			ContactIDs:      []int{1},
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
			createContactFunc: func(ctx context.Context, q DBTX, campaignID int, contactID int) error {
				return nil
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

		badTime := "not-a-time"
		_, err := svc.Create(context.Background(), CreateCampaignRequest{
			Name:            "Bad",
			MessageTemplate: "Hi",
			ChannelID:       1,
			ContactIDs:      []int{1},
			ScheduledAt:     &badTime,
		}, testTenant(), 1)
		if err == nil {
			t.Fatal("expected error for invalid scheduled_at format")
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

	t.Run("Send updates status to sending then processes contacts", func(t *testing.T) {
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
			getPendingFunc: func(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignContact, error) {
				return nil, nil
			},
			countPendingFunc: func(ctx context.Context, q DBTX, campaignID int) (int, error) {
				return 0, nil
			},
		}
		svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}

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
				svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

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
		svc := &campaignService{repo: repo, db: nil, cfg: testConfig(), logger: testLogger()}

		err := svc.Send(context.Background(), 1, 999)
		if err == nil {
			t.Fatal("expected error for non-existent campaign")
		}
	})
}
