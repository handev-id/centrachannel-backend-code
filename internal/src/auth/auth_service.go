package auth

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"centrachannel/config"
	"centrachannel/internal/src/tenant"
	"centrachannel/internal/utils/hash"
	"centrachannel/internal/utils/logger"
)

type AuthService interface {
	Register(ctx context.Context, req RegisterRequest, t *tenant.Tenant) (*User, error)
	Login(ctx context.Context, req LoginRequest, t *tenant.Tenant) (string, error)
	CheckToken(ctx context.Context, token string, t *tenant.Tenant) (*User, error)
	Logout(ctx context.Context, tokenStr string) error
}

type authService struct {
	repo   AuthRepository
	db     *sql.DB
	cfg    *config.Config
	logger *logger.Logger
	rdb    *redis.Client
}

func NewAuthService(repo AuthRepository, db *sql.DB, cfg *config.Config, logger *logger.Logger, rdb *redis.Client) AuthService {
	return &authService{repo: repo, db: db, cfg: cfg, logger: logger, rdb: rdb}
}

func (s *authService) Register(ctx context.Context, req RegisterRequest, t *tenant.Tenant) (*User, error) {
	exists, err := s.repo.GetByEmail(ctx, s.db, t.ID, req.Email)
	if err != nil {
		return nil, err
	}
	if exists != nil {
		return nil, fmt.Errorf("email already registered")
	}

	existsUsername, err := s.repo.GetByUsername(ctx, s.db, t.ID, req.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to check username: %w", err)
	}
	if existsUsername != nil {
		return nil, fmt.Errorf("username already taken")
	}

	hashed, err := hash.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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

	userID, err := s.repo.Create(ctx, tx, user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	var agentRoleID int
	err = tx.QueryRowContext(ctx, `SELECT id FROM roles WHERE tenant_id = $1 AND name = 'agent'`, t.ID).Scan(&agentRoleID)
	if err != nil {
		return nil, fmt.Errorf("failed to find agent role: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT INTO role_user (tenant_id, user_id, role_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`, t.ID, userID, agentRoleID, time.Now(), time.Now()); err != nil {
		return nil, fmt.Errorf("failed to attach role: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	user.ID = userID
	user.Roles = []Role{{ID: agentRoleID, TenantID: t.ID, Name: "agent"}}
	return user, nil
}

func (s *authService) Login(ctx context.Context, req LoginRequest, t *tenant.Tenant) (string, error) {
	user, err := s.repo.GetByUsername(ctx, s.db, t.ID, req.Username)
	if err != nil {
		s.logger.Error("login lookup failed: %v", err)
		return "", fmt.Errorf("invalid credentials")
	}
	if user == nil {
		return "", fmt.Errorf("invalid credentials")
	}

	if !hash.VerifyPassword(user.Password, req.Password) {
		return "", fmt.Errorf("invalid credentials")
	}

	_, _ = s.db.ExecContext(ctx, `UPDATE users SET last_login = $1 WHERE id = $2 AND tenant_id = $3`, time.Now(), user.ID, t.ID)

	roles, err := s.repo.GetRolesByUserID(ctx, s.db, t.ID, user.ID)
	if err != nil {
		return "", fmt.Errorf("failed to load user roles: %w", err)
	}

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	now := time.Now()
	claims := jwt.MapClaims{
		"jti":    uuid.New().String(),
		"sub":    user.ID,
		"tenant": t.ID,
		"domain": t.Domain,
		"iat":    now.Unix(),
		"exp":    now.Add(s.cfg.JWTExpiry).Unix(),
		"user":   user.Username,
		"roles":  roleNames,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func (s *authService) Logout(ctx context.Context, tokenStr string) error {
	token, err := jwt.Parse(tokenStr, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("invalid token claims")
	}

	jti, _ := claims["jti"].(string)
	if jti == "" {
		return nil
	}

	if s.rdb == nil {
		return nil
	}

	expFloat, _ := claims["exp"].(float64)
	ttl := time.Unix(int64(expFloat), 0).Sub(time.Now())
	if ttl <= 0 {
		return nil
	}

	return s.rdb.Set(ctx, "token_blacklist:"+jti, "1", ttl).Err()
}

func (s *authService) CheckToken(ctx context.Context, tokenStr string, t *tenant.Tenant) (*User, error) {
	token, err := jwt.Parse(tokenStr, func(tk *jwt.Token) (interface{}, error) {
		if _, ok := tk.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if jti, _ := claims["jti"].(string); jti != "" && s.rdb != nil {
		blacklisted, err := s.rdb.Exists(ctx, "token_blacklist:"+jti).Result()
		if err == nil && blacklisted > 0 {
			return nil, fmt.Errorf("token has been revoked")
		}
	}

	subFloat, ok := claims["sub"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid sub claim")
	}
	userID := int(subFloat)

	user, err := s.repo.GetByID(ctx, s.db, t.ID, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}

	roles, err := s.repo.GetRolesByUserID(ctx, s.db, t.ID, userID)
	if err == nil {
		user.Roles = roles
	}

	return user, nil
}
