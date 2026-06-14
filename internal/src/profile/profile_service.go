package profile

import (
	"context"
	"database/sql"
)

type ProfileService interface {
	List(ctx context.Context, q ListProfileQuery) ([]Profile, error)
	GetByID(ctx context.Context, id int) (*Profile, error)
	Update(ctx context.Context, id int, req UpdateProfileRequest) (*Profile, error)
}

type profileService struct {
	repo ProfileRepository
	db   *sql.DB
}

func NewProfileService(repo ProfileRepository, db *sql.DB) ProfileService {
	return &profileService{repo: repo, db: db}
}

func (s *profileService) List(ctx context.Context, q ListProfileQuery) ([]Profile, error) {
	return s.repo.List(ctx, s.db, q.ContactID, q.ChannelID)
}

func (s *profileService) GetByID(ctx context.Context, id int) (*Profile, error) {
	return s.repo.GetByID(ctx, s.db, id)
}

func (s *profileService) Update(ctx context.Context, id int, req UpdateProfileRequest) (*Profile, error) {
	existing, err := s.repo.GetByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}

	if req.IsMain != nil { existing.IsMain = *req.IsMain }
	if req.LinkedDeviceWhatsappID != nil { existing.LinkedDeviceWhatsappID = req.LinkedDeviceWhatsappID }
	if req.DisplayName != nil { existing.DisplayName = req.DisplayName }

	if err := s.repo.Update(ctx, s.db, id, existing); err != nil {
		return nil, err
	}

	return existing, nil
}
