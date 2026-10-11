package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"centrachannel/internal/utils"
)

type authRepository struct{}

func NewAuthRepository() AuthRepository {
	return &authRepository{}
}

func scanUser(row interface {
	Scan(dest ...interface{}) error
}) (*User, error) {
	var u User
	var lastName, phone sql.NullString
	var avatar sql.NullString
	var lastLogin sql.NullTime
	var deletedAt utils.NullableTime
	var tenantID uuid.UUID

	err := row.Scan(&u.ID, &tenantID, &u.FirstName, &lastName, &u.Username, &u.Email, &phone, &u.Password, &avatar, &lastLogin, &deletedAt, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.TenantID = tenantID
	if lastName.Valid {
		u.LastName = &lastName.String
	}
	if phone.Valid {
		u.Phone = &phone.String
	}
	if avatar.Valid {
		u.Avatar = json.RawMessage(avatar.String)
	}
	if lastLogin.Valid {
		u.LastLogin = &lastLogin.Time
	}
	u.DeletedAt = deletedAt
	return &u, nil
}

func scanRole(rows interface {
	Scan(dest ...interface{}) error
}) (Role, error) {
	var r Role
	err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (r *authRepository) GetByEmail(ctx context.Context, q DBTX, tenantID uuid.UUID, email string) (*User, error) {
	return scanUser(q.QueryRowContext(ctx, `SELECT id, tenant_id, first_name, last_name, username, email, phone, password, avatar, last_login, deleted_at, created_at, updated_at FROM users WHERE email = $1 AND tenant_id = $2 AND deleted_at IS NULL`, email, tenantID))
}

func (r *authRepository) GetByUsername(ctx context.Context, q DBTX, tenantID uuid.UUID, username string) (*User, error) {
	return scanUser(q.QueryRowContext(ctx, `SELECT id, tenant_id, first_name, last_name, username, email, phone, password, avatar, last_login, deleted_at, created_at, updated_at FROM users WHERE username = $1 AND tenant_id = $2 AND deleted_at IS NULL`, username, tenantID))
}

func (r *authRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*User, error) {
	return scanUser(q.QueryRowContext(ctx, `SELECT id, tenant_id, first_name, last_name, username, email, phone, password, avatar, last_login, deleted_at, created_at, updated_at FROM users WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`, id, tenantID))
}

func (r *authRepository) Create(ctx context.Context, db *sql.DB, user *User, roleName string) (uuid.UUID, uuid.UUID, error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	defer tx.Rollback()
	var avatarBytes []byte
	if user.Avatar != nil {
		avatarBytes = user.Avatar
	}
	var userID uuid.UUID
	err = tx.QueryRowContext(ctx, `INSERT INTO users (tenant_id, first_name, last_name, username, email, phone, password, avatar, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`,
		user.TenantID, user.FirstName, user.LastName, user.Username, user.Email, user.Phone,
		user.Password, avatarBytes, time.Now(), time.Now(),
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	roleID, err := r.GetRoleIDByName(ctx, tx, user.TenantID, roleName)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if err := r.AttachRole(ctx, tx, user.TenantID, userID, roleID); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	if err := tx.Commit(); err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return userID, roleID, nil
}

func (r *authRepository) GetRolesByUserID(ctx context.Context, q DBTX, tenantID uuid.UUID, userID uuid.UUID) ([]Role, error) {
	rows, err := q.QueryContext(ctx, `SELECT r.id, r.tenant_id, r.name, r.created_at, r.updated_at FROM roles r INNER JOIN role_user ru ON ru.role_id = r.id WHERE ru.user_id = $1 AND r.tenant_id = $2`, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		role, err := scanRole(rows)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *authRepository) GetRoleIDByName(ctx context.Context, q DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRowContext(ctx, `SELECT id FROM roles WHERE tenant_id = $1 AND name = $2`, tenantID, name).Scan(&id)
	return id, err
}
func (r *authRepository) AttachRole(ctx context.Context, q DBTX, tenantID, userID, roleID uuid.UUID) error {
	_, err := q.ExecContext(ctx, `INSERT INTO role_user (tenant_id, user_id, role_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`, tenantID, userID, roleID, time.Now(), time.Now())
	return err
}
func (r *authRepository) UpdateLastLogin(ctx context.Context, q DBTX, tenantID, userID uuid.UUID) error {
	_, err := q.ExecContext(ctx, `UPDATE users SET last_login = $1 WHERE id = $2 AND tenant_id = $3`, time.Now(), userID, tenantID)
	return err
}
