package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"centrachannel/config"
	"centrachannel/internal/utils/logger"
)

type TenantService interface {
	GetDomains(ctx context.Context, id uuid.UUID) (json.RawMessage, error)
	GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	Update(ctx context.Context, tenant *Tenant) error
}

type tenantService struct {
	repo   TenantRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewTenantService(repo TenantRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) TenantService {
	return &tenantService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *tenantService) GetByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	return s.repo.GetByID(ctx, s.db, id)
}

func (s *tenantService) Update(ctx context.Context, tenant *Tenant) error {
	tenant.UpdatedAt = time.Now()
	return s.repo.Update(ctx, s.db, tenant)
}

type TenantResolver interface {
	Resolve(context.Context, string) (*Tenant, error)
}

type tenantResolver struct {
	repo   TenantRepository
	db     *sql.DB
	redis  *redis.Client
	logger *logger.Logger
}

func NewTenantResolver(repo TenantRepository, db *sql.DB, rdb *redis.Client, log *logger.Logger) TenantResolver {
	return &tenantResolver{repo: repo, db: db, redis: rdb, logger: log}
}
func (s *tenantResolver) Resolve(ctx context.Context, name string) (*Tenant, error) {
	name = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(name)), ".")
	id, err := s.repo.GetActiveIDByDomain(ctx, s.db, name)
	if err != nil {
		return nil, err
	}
	key := "tenant:v2:" + id.String()
	if s.redis != nil {
		data, cacheErr := s.redis.Get(ctx, key).Bytes()
		if cacheErr == nil {
			var t Tenant
			if json.Unmarshal(data, &t) == nil && t.ID == id && t.IsActive {
				return &t, nil
			}
		} else if cacheErr != redis.Nil {
			s.logger.Error("tenant cache lookup failed: %v", cacheErr)
		}
	}
	t, err := s.repo.GetByID(ctx, s.db, id)
	if err != nil {
		return nil, err
	}
	if !t.IsActive {
		return nil, sql.ErrNoRows
	}
	if s.redis != nil {
		if data, err := json.Marshal(t); err == nil {
			if err := s.redis.Set(ctx, key, data, time.Hour).Err(); err != nil {
				s.logger.Error("tenant cache write failed: %v", err)
			}
		}
	}
	return t, nil
}

func (s *tenantService) GetDomains(ctx context.Context, id uuid.UUID) (json.RawMessage, error) {
	return s.repo.GetDomains(ctx, s.db, id)
}
