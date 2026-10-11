package campaign

import (
	"context"
	"database/sql/driver"
	"testing"

	"github.com/google/uuid"
)

// Scoped lookups deliberately return nil, nil for IDs outside the tenant as well
// as nonexistent IDs. Neither result grants permission to use the resource.
func TestCampaignRejectsUnavailableRecipientList(t *testing.T) {
	listID := uuid.New()
	operations := map[string]func(*campaignService) error{
		"create campaign": func(s *campaignService) error {
			_, err := s.Create(context.Background(), CreateCampaignRequest{Name: "Campaign", RecipientListID: &listID}, testTenant(), uuid.New())
			return err
		},
		"update campaign": func(s *campaignService) error {
			_, err := s.Update(context.Background(), testTenant().ID, uuid.New(), UpdateCampaignRequest{RecipientListID: &listID})
			return err
		},
		"add recipient": func(s *campaignService) error {
			_, err := s.AddRecipientContact(context.Background(), testTenant().ID, listID, AddContactToListRequest{FirstName: "Unauthorized"})
			return err
		},
		"list recipients": func(s *campaignService) error {
			_, err := s.ListRecipientContacts(context.Background(), testTenant().ID, listID)
			return err
		},
		"remove recipient": func(s *campaignService) error {
			return s.RemoveRecipientContact(context.Background(), testTenant().ID, listID, uuid.New())
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			touched := false
			repo := &mockCampaignRepository{
				getRecipientListByIDFunc: func(ctx context.Context, q DBTX, tenantID, id uuid.UUID) (*CampaignRecipientList, error) {
					if tenantID != testTenant().ID || id != listID {
						t.Fatalf("lookup lost tenant or list ID")
					}
					return nil, nil
				},
				createFunc: func(context.Context, DBTX, *Campaign) (uuid.UUID, error) { touched = true; return uuid.New(), nil },
				updateFunc: func(context.Context, DBTX, uuid.UUID, uuid.UUID, *Campaign) error { touched = true; return nil },
				createRecipientContactFunc: func(context.Context, DBTX, *CampaignRecipientContact) (uuid.UUID, error) {
					touched = true
					return uuid.New(), nil
				},
				listRecipientContactsFunc: func(context.Context, DBTX, uuid.UUID) ([]CampaignRecipientContact, error) {
					touched = true
					return nil, nil
				},
				deleteRecipientContactFunc: func(context.Context, DBTX, uuid.UUID) error { touched = true; return nil },
			}
			db := newMockDB(func() (driver.Tx, error) { touched = true; return &mockTx{}, nil })
			defer db.Close()
			svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}
			if err := operation(svc); err == nil {
				t.Error("unavailable recipient list must return an error")
			}
			if touched {
				t.Error("unavailable recipient list reached transaction or dependent repository operation")
			}
		})
	}
}

func TestCampaignRejectsUnavailableTemplate(t *testing.T) {
	templateID := uuid.New()
	operations := map[string]func(*campaignService) error{
		"create campaign": func(s *campaignService) error {
			_, err := s.Create(context.Background(), CreateCampaignRequest{Name: "Campaign", TemplateID: &templateID}, testTenant(), uuid.New())
			return err
		},
		"update campaign": func(s *campaignService) error {
			_, err := s.Update(context.Background(), testTenant().ID, uuid.New(), UpdateCampaignRequest{TemplateID: &templateID})
			return err
		},
	}
	for name, operation := range operations {
		t.Run(name, func(t *testing.T) {
			touched := false
			repo := &mockCampaignRepository{
				getTemplateByIDFunc: func(ctx context.Context, q DBTX, tenantID, id uuid.UUID) (*CampaignTemplate, error) {
					if tenantID != testTenant().ID || id != templateID {
						t.Fatalf("lookup lost tenant or template ID")
					}
					return nil, nil
				},
				createFunc: func(context.Context, DBTX, *Campaign) (uuid.UUID, error) { touched = true; return uuid.New(), nil },
				updateFunc: func(context.Context, DBTX, uuid.UUID, uuid.UUID, *Campaign) error { touched = true; return nil },
			}
			db := newMockDB(func() (driver.Tx, error) { touched = true; return &mockTx{}, nil })
			defer db.Close()
			svc := &campaignService{repo: repo, db: db, cfg: testConfig(), logger: testLogger()}
			if err := operation(svc); err == nil {
				t.Error("unavailable template must return an error")
			}
			if touched {
				t.Error("unavailable template reached transaction or write")
			}
		})
	}
}
