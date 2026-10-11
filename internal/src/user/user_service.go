package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/hash"
	"centrachannel/internal/utils/logger"
)

type OnlineChecker interface {
	IsUserOnline(tenantID uuid.UUID, userID uuid.UUID) bool
}

type UserService interface {
	ListRoles(ctx context.Context, tenantID uuid.UUID) ([]Role, error)
	List(ctx context.Context, q ListUserQuery, t *tenant.Tenant) ([]*User, int, error)
	GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*User, error)
	Create(ctx context.Context, req CreateUserRequest, t *tenant.Tenant) (*User, error)
	Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error
}

type userService struct {
	repo    UserRepository
	db      *sql.DB
	cfg     *config.Config
	logger  *logger.Logger
	checker OnlineChecker
}

func NewUserService(repo UserRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, checker ...OnlineChecker) UserService {
	s := &userService{repo: repo, db: db, cfg: cfg, logger: logger}
	if len(checker) > 0 {
		s.checker = checker[0]
	}
	return s
}

func (s *userService) ListRoles(ctx context.Context, tenantID uuid.UUID) ([]Role, error) {
	return s.repo.ListRoles(ctx, s.db, tenantID)
}

func (s *userService) List(ctx context.Context, q ListUserQuery, t *tenant.Tenant) ([]*User, int, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	offset := (q.Page - 1) * q.Limit

	users, total, err := s.repo.List(ctx, s.db, t.ID, q.Limit, offset, q.Search, q.RoleID)
	if err != nil {
		return nil, 0, err
	}

	if len(users) > 0 {
		userIDs := make([]uuid.UUID, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}
		rolesMap, _ := s.repo.GetRolesByUserIDs(ctx, s.db, t.ID, userIDs)
		for _, u := range users {
			u.Roles = rolesMap[u.ID]
			if s.checker != nil {
				u.IsOnline = s.checker.IsUserOnline(t.ID, u.ID)
			}
		}
	}

	return users, total, nil
}

func (s *userService) GetByID(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*User, error) {
	user, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	roles, _ := s.repo.GetRolesByUserID(ctx, s.db, tenantID, user.ID)
	user.Roles = roles
	if s.checker != nil {
		user.IsOnline = s.checker.IsUserOnline(tenantID, user.ID)
	}
	return user, nil
}

func (s *userService) Create(ctx context.Context, req CreateUserRequest, t *tenant.Tenant) (*User, error) {
	hashed, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user := &User{
		TenantID:  t.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  hashed,
		Avatar:    req.Avatar,
	}

	userID, err := s.repo.Create(ctx, s.db, user, req.Roles)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = userID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	roles, _ := s.repo.GetRolesByUserID(ctx, s.db, t.ID, userID)
	user.Roles = roles
	return user, nil
}

func (s *userService) Update(ctx context.Context, tenantID uuid.UUID, id uuid.UUID, req UpdateUserRequest) (*User, error) {
	user, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}

	hashed := user.Password
	if req.Password != nil {
		hashed, err = hash.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
	}

	updated := &User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  hashed,
		Avatar:    req.Avatar,
	}

	if err := s.repo.Update(ctx, s.db, tenantID, id, updated, req.Roles); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return s.GetByID(ctx, tenantID, id)
}

func (s *userService) Delete(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	if _, err := s.repo.GetByID(ctx, s.db, tenantID, id); err != nil {
		return err
	}
	roles, err := s.repo.GetRolesByUserID(ctx, s.db, tenantID, id)
	if err != nil {
		return err
	}
	for _, role := range roles {
		if role.Name == "super-admin" {
			return fmt.Errorf("You can not delete super admin")
		}
	}
	return s.repo.SoftDelete(ctx, s.db, tenantID, id)
}
