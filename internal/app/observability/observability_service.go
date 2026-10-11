package observability

import (
	"context"
	"database/sql"

	"centrachannel/config"
	"centrachannel/internal/utils/hostname"
)

type ObservabilityService interface {
	AllowTLS(context.Context, string) (bool, error)
}
type observabilityService struct {
	repo ObservabilityRepository
	db   *sql.DB
	cfg  *config.Config
}

func NewObservabilityService(repo ObservabilityRepository, db *sql.DB, cfg *config.Config) ObservabilityService {
	return &observabilityService{repo: repo, db: db, cfg: cfg}
}
func (s *observabilityService) AllowTLS(ctx context.Context, name string) (bool, error) {
	name, err := hostname.NormalizeName(name)
	if err != nil {
		return false, nil
	}
	base, err := hostname.NormalizeName(s.cfg.BaseDomain)
	if err == nil && name == base {
		return true, nil
	}
	return s.repo.IsActiveDomain(ctx, s.db, name)
}
