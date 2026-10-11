package domain

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/utils/hostname"
)

var (
	ErrDefault  = errors.New("Move the default domain before disabling or deleting it")
	ErrConflict = errors.New("Domain name is already registered")
)

type DomainService interface {
	List(ctx context.Context, tenantID uuid.UUID) ([]Domain, error)
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*Domain, error)
	Create(ctx context.Context, req CreateDomainRequest, tenantID uuid.UUID) (*Domain, error)
	Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateDomainRequest) (*Domain, error)
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type domainService struct {
	repo DomainRepository
	db   *sql.DB
	cfg  *config.Config
}

func NewDomainService(repo DomainRepository, db *sql.DB, cfg *config.Config) DomainService {
	return &domainService{repo: repo, db: db, cfg: cfg}
}

func (s *domainService) List(ctx context.Context, tenantID uuid.UUID) ([]Domain, error) {
	return s.repo.List(ctx, s.db, tenantID)
}
func (s *domainService) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*Domain, error) {
	return s.repo.GetByID(ctx, s.db, tenantID, id)
}
func (s *domainService) Create(ctx context.Context, req CreateDomainRequest, tenantID uuid.UUID) (*Domain, error) {
	name, err := hostname.ValidateName(req.Name, s.cfg.BaseDomain)
	if err != nil {
		return nil, err
	}
	d := &Domain{TenantID: tenantID, Name: name, IsActive: true}
	if req.IsDefault != nil {
		d.IsDefault = *req.IsDefault
	}
	if req.IsActive != nil {
		d.IsActive = *req.IsActive
	}
	if d.IsDefault && !d.IsActive {
		return nil, ErrDefault
	}
	id, err := s.repo.Create(ctx, s.db, d)
	if err != nil {
		return nil, err
	}
	d.ID = id
	return d, nil
}
func (s *domainService) Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateDomainRequest) (*Domain, error) {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		name, err := hostname.ValidateName(*req.Name, s.cfg.BaseDomain)
		if err != nil {
			return nil, err
		}
		existing.Name = name
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.IsDefault != nil {
		if existing.IsDefault && !*req.IsDefault {
			return nil, ErrDefault
		}
		existing.IsDefault = *req.IsDefault
	}
	if existing.IsDefault && !existing.IsActive {
		return nil, ErrDefault
	}
	if err := s.repo.Update(ctx, s.db, tenantID, id, existing); err != nil {
		return nil, err
	}
	return existing, nil
}
func (s *domainService) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	existing, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return err
	}
	if existing.IsDefault {
		return ErrDefault
	}
	return s.repo.Delete(ctx, s.db, tenantID, id)
}
