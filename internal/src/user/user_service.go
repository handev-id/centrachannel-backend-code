package user

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/avatar"
	"centrachannel/internal/utils/hash"
	"centrachannel/internal/utils/logger"
)

type UserService interface {
	List(ctx context.Context, q ListUserQuery, t *tenant.Tenant) (*PaginatedResponse, error)
	GetByID(ctx context.Context, tenantID int, id int) (*User, error)
	Create(ctx context.Context, req CreateUserRequest, t *tenant.Tenant) (*User, error)
	Update(ctx context.Context, tenantID int, id int, req UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, tenantID int, id int) error
}

type userService struct {
	repo   UserRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
}

func NewUserService(repo UserRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger) UserService {
	return &userService{repo: repo, db: db, cfg: cfg, logger: logger}
}

func (s *userService) List(ctx context.Context, q ListUserQuery, t *tenant.Tenant) (*PaginatedResponse, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.Limit < 1 || q.Limit > 100 {
		q.Limit = 20
	}

	offset := (q.Page - 1) * q.Limit

	users, total, err := s.repo.List(ctx, s.db, t.ID, q.Limit, offset, q.Search, q.RoleID)
	if err != nil {
		return nil, err
	}

	if len(users) > 0 {
		userIDs := make([]int, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}
		rolesMap, _ := s.repo.GetRolesByUserIDs(ctx, s.db, t.ID, userIDs)
		for _, u := range users {
			u.Roles = rolesMap[u.ID]
		}
	}

	lastPage := int(math.Ceil(float64(total) / float64(q.Limit)))
	from := offset + 1
	to := offset + len(users)
	if to > total {
		to = total
	}
	if total == 0 {
		from = 0
		to = 0
	}

	meta := PaginationMeta{
		Total:       total,
		PerPage:     q.Limit,
		CurrentPage: q.Page,
		LastPage:    lastPage,
		From:        from,
		To:          to,
	}

	return &PaginatedResponse{Meta: meta, Data: users}, nil
}

func (s *userService) GetByID(ctx context.Context, tenantID int, id int) (*User, error) {
	user, err := s.repo.GetByID(ctx, s.db, tenantID, id)
	if err != nil {
		return nil, err
	}
	roles, _ := s.repo.GetRolesByUserID(ctx, s.db, tenantID, user.ID)
	user.Roles = roles
	return user, nil
}

func (s *userService) Create(ctx context.Context, req CreateUserRequest, t *tenant.Tenant) (*User, error) {
	password := req.Username
	if req.Password != nil {
		password = *req.Password
	}

	hashed, err := hash.HashPassword(password)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	avatarData := req.Avatar
	if len(avatarData) == 0 {
		avatarData = avatar.GenerateInitials(req.FirstName)
	}

	user := &User{
		TenantID:  t.ID,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  hashed,
		Avatar:    avatarData,
	}

	userID, err := s.repo.Create(ctx, tx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	if err := s.repo.AttachRoles(ctx, tx, t.ID, userID, req.Roles); err != nil {
		return nil, fmt.Errorf("failed to attach roles: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	user.ID = userID
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()
	roles, _ := s.repo.GetRolesByUserID(ctx, s.db, t.ID, userID)
	user.Roles = roles
	return user, nil
}

func (s *userService) Update(ctx context.Context, tenantID int, id int, req UpdateUserRequest) (*User, error) {
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

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	updated := &User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Username:  req.Username,
		Email:     req.Email,
		Phone:     req.Phone,
		Password:  hashed,
		Avatar:    req.Avatar,
	}

	if err := s.repo.Update(ctx, tx, tenantID, id, updated); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	if err := s.repo.SyncRoles(ctx, tx, tenantID, id, req.Roles); err != nil {
		return nil, fmt.Errorf("failed to sync roles: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return s.GetByID(ctx, tenantID, id)
}

func (s *userService) Delete(ctx context.Context, tenantID int, id int) error {
	if id == 1 {
		return fmt.Errorf("forbidden: you can not delete super admin")
	}
	return s.repo.SoftDelete(ctx, s.db, tenantID, id)
}
