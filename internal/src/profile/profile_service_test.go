package profile

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

type mockDB struct{}

func (m *mockDB) ExecContext(_ context.Context, _ string, _ ...interface{}) (sql.Result, error) {
	return nil, nil
}

func (m *mockDB) QueryContext(_ context.Context, _ string, _ ...interface{}) (*sql.Rows, error) {
	return nil, nil
}

func (m *mockDB) QueryRowContext(_ context.Context, _ string, _ ...interface{}) *sql.Row {
	return nil
}

type mockProfileRepository struct {
	listFn           func(ctx context.Context, q DBTX, contactID, channelID int) ([]Profile, error)
	getByIDFn        func(ctx context.Context, q DBTX, id int) (*Profile, error)
	updateFn         func(ctx context.Context, q DBTX, id int, profile *Profile) error
	getByContactIDFn func(ctx context.Context, q DBTX, contactID int) ([]Profile, error)
}

func (m *mockProfileRepository) List(ctx context.Context, q DBTX, contactID, channelID int) ([]Profile, error) {
	if m.listFn == nil {
		return nil, nil
	}
	return m.listFn(ctx, q, contactID, channelID)
}

func (m *mockProfileRepository) GetByID(ctx context.Context, q DBTX, id int) (*Profile, error) {
	if m.getByIDFn == nil {
		return nil, nil
	}
	return m.getByIDFn(ctx, q, id)
}

func (m *mockProfileRepository) Update(ctx context.Context, q DBTX, id int, profile *Profile) error {
	if m.updateFn == nil {
		return nil
	}
	return m.updateFn(ctx, q, id, profile)
}

func (m *mockProfileRepository) GetByContactID(ctx context.Context, q DBTX, contactID int) ([]Profile, error) {
	if m.getByContactIDFn == nil {
		return nil, nil
	}
	return m.getByContactIDFn(ctx, q, contactID)
}

func baseProfile(id, contactID, channelID int) Profile {
	now := time.Now()
	return Profile{
		ID:                     id,
		ExternalID:             "ext-123",
		DisplayName:            strPtr("Original Name"),
		IsMain:                 true,
		LinkedDeviceWhatsappID: strPtr("wa-original"),
		ContactID:              contactID,
		ChannelID:              channelID,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
}

func strPtr(s string) *string { return &s }

func boolPtr(b bool) *bool { return &b }

func TestProfileServiceList(t *testing.T) {
	t.Run("filters by contact_id and channel_id", func(t *testing.T) {
		wantContactID := 10
		wantChannelID := 20
		profiles := []Profile{
			baseProfile(1, wantContactID, wantChannelID),
			baseProfile(2, wantContactID, wantChannelID),
		}
		repo := &mockProfileRepository{
			listFn: func(_ context.Context, _ DBTX, contactID, channelID int) ([]Profile, error) {
				if contactID != wantContactID {
					t.Errorf("expected contactID %d, got %d", wantContactID, contactID)
				}
				if channelID != wantChannelID {
					t.Errorf("expected channelID %d, got %d", wantChannelID, channelID)
				}
				return profiles, nil
			},
		}
		svc := NewProfileService(repo, nil)

		result, err := svc.List(context.Background(), ListProfileQuery{
			ContactID: wantContactID,
			ChannelID: wantChannelID,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 2 {
			t.Fatalf("expected 2 profiles, got %d", len(result))
		}
		if result[0].ID != 1 || result[1].ID != 2 {
			t.Errorf("expected profiles with IDs 1 and 2, got %+v", result)
		}
	})

	t.Run("empty list when no match", func(t *testing.T) {
		repo := &mockProfileRepository{
			listFn: func(_ context.Context, _ DBTX, _, _ int) ([]Profile, error) {
				return []Profile{}, nil
			},
		}
		svc := NewProfileService(repo, nil)

		result, err := svc.List(context.Background(), ListProfileQuery{ContactID: 99, ChannelID: 99})
		if err != nil {
			t.Fatal(err)
		}
		if len(result) != 0 {
			t.Fatalf("expected 0 profiles, got %d", len(result))
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockProfileRepository{
			listFn: func(_ context.Context, _ DBTX, _, _ int) ([]Profile, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewProfileService(repo, nil)

		_, err := svc.List(context.Background(), ListProfileQuery{ContactID: 1, ChannelID: 1})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestProfileServiceGetByID(t *testing.T) {
	t.Run("returns profile", func(t *testing.T) {
		want := baseProfile(42, 1, 2)
		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, id int) (*Profile, error) {
				if id != 42 {
					t.Errorf("expected id 42, got %d", id)
				}
				return &want, nil
			},
		}
		svc := NewProfileService(repo, nil)

		result, err := svc.GetByID(context.Background(), 42)
		if err != nil {
			t.Fatal(err)
		}
		if result.ID != 42 {
			t.Errorf("expected ID 42, got %d", result.ID)
		}
		if result.ContactID != 1 || result.ChannelID != 2 {
			t.Errorf("unexpected profile: %+v", result)
		}
	})

	t.Run("nil when not found", func(t *testing.T) {
		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, _ int) (*Profile, error) {
				return nil, nil
			},
		}
		svc := NewProfileService(repo, nil)

		result, err := svc.GetByID(context.Background(), 999)
		if err != nil {
			t.Fatal(err)
		}
		if result != nil {
			t.Fatalf("expected nil, got %+v", result)
		}
	})

	t.Run("repository error", func(t *testing.T) {
		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, _ int) (*Profile, error) {
				return nil, errors.New("db error")
			},
		}
		svc := NewProfileService(repo, nil)

		_, err := svc.GetByID(context.Background(), 1)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestProfileServiceUpdate(t *testing.T) {
	t.Run("partial update: IsMain", func(t *testing.T) {
		existing := baseProfile(1, 10, 20)
		existing.IsMain = true
		var captured *Profile

		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, id int) (*Profile, error) {
				return &existing, nil
			},
			updateFn: func(_ context.Context, _ DBTX, id int, p *Profile) error {
				captured = p
				return nil
			},
		}
		svc := NewProfileService(repo, nil)

		falseVal := false
		result, err := svc.Update(context.Background(), 1, UpdateProfileRequest{IsMain: &falseVal})
		if err != nil {
			t.Fatal(err)
		}
		if result.IsMain != false {
			t.Errorf("expected IsMain=false, got %v", result.IsMain)
		}
		if captured == nil || captured.IsMain != false {
			t.Errorf("expected captured IsMain=false, got %v", captured.IsMain)
		}
	})

	t.Run("partial update: DisplayName", func(t *testing.T) {
		existing := baseProfile(2, 10, 20)
		var captured *Profile

		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, id int) (*Profile, error) {
				return &existing, nil
			},
			updateFn: func(_ context.Context, _ DBTX, id int, p *Profile) error {
				captured = p
				return nil
			},
		}
		svc := NewProfileService(repo, nil)

		newName := "Updated Name"
		result, err := svc.Update(context.Background(), 2, UpdateProfileRequest{DisplayName: &newName})
		if err != nil {
			t.Fatal(err)
		}
		if result.DisplayName == nil || *result.DisplayName != "Updated Name" {
			t.Errorf("expected DisplayName='Updated Name', got %v", result.DisplayName)
		}
		if captured == nil || captured.DisplayName == nil || *captured.DisplayName != "Updated Name" {
			t.Errorf("expected captured DisplayName='Updated Name', got %v", captured.DisplayName)
		}
	})

	t.Run("partial update: LinkedDeviceWhatsappID", func(t *testing.T) {
		existing := baseProfile(3, 10, 20)
		var captured *Profile

		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, id int) (*Profile, error) {
				return &existing, nil
			},
			updateFn: func(_ context.Context, _ DBTX, id int, p *Profile) error {
				captured = p
				return nil
			},
		}
		svc := NewProfileService(repo, nil)

		newWa := "wa-new-device"
		result, err := svc.Update(context.Background(), 3, UpdateProfileRequest{LinkedDeviceWhatsappID: &newWa})
		if err != nil {
			t.Fatal(err)
		}
		if result.LinkedDeviceWhatsappID == nil || *result.LinkedDeviceWhatsappID != "wa-new-device" {
			t.Errorf("expected LinkedDeviceWhatsappID='wa-new-device', got %v", result.LinkedDeviceWhatsappID)
		}
		if captured == nil || captured.LinkedDeviceWhatsappID == nil || *captured.LinkedDeviceWhatsappID != "wa-new-device" {
			t.Errorf("expected captured LinkedDeviceWhatsappID='wa-new-device', got %v", captured.LinkedDeviceWhatsappID)
		}
	})

	t.Run("no changes when all fields nil", func(t *testing.T) {
		existing := baseProfile(4, 10, 20)
		var captured *Profile

		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, id int) (*Profile, error) {
				return &existing, nil
			},
			updateFn: func(_ context.Context, _ DBTX, id int, p *Profile) error {
				captured = p
				return nil
			},
		}
		svc := NewProfileService(repo, nil)

		result, err := svc.Update(context.Background(), 4, UpdateProfileRequest{})
		if err != nil {
			t.Fatal(err)
		}
		if captured == nil {
			t.Fatal("expected update to be called")
		}
		// Original values should be preserved
		if captured.IsMain != true {
			t.Errorf("expected IsMain=true, got %v", captured.IsMain)
		}
		if captured.DisplayName == nil || *captured.DisplayName != "Original Name" {
			t.Errorf("expected DisplayName='Original Name', got %v", captured.DisplayName)
		}
		if captured.LinkedDeviceWhatsappID == nil || *captured.LinkedDeviceWhatsappID != "wa-original" {
			t.Errorf("expected LinkedDeviceWhatsappID='wa-original', got %v", captured.LinkedDeviceWhatsappID)
		}
		// Result should match
		if result.IsMain != true {
			t.Errorf("expected result IsMain=true, got %v", result.IsMain)
		}
	})

	t.Run("error when GetByID fails", func(t *testing.T) {
		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, _ int) (*Profile, error) {
				return nil, errors.New("not found")
			},
		}
		svc := NewProfileService(repo, nil)

		_, err := svc.Update(context.Background(), 999, UpdateProfileRequest{IsMain: boolPtr(false)})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("error when Update fails", func(t *testing.T) {
		existing := baseProfile(1, 10, 20)
		repo := &mockProfileRepository{
			getByIDFn: func(_ context.Context, _ DBTX, _ int) (*Profile, error) {
				return &existing, nil
			},
			updateFn: func(_ context.Context, _ DBTX, _ int, _ *Profile) error {
				return errors.New("update failed")
			},
		}
		svc := NewProfileService(repo, nil)

		_, err := svc.Update(context.Background(), 1, UpdateProfileRequest{DisplayName: strPtr("x")})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
