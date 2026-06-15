package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"centrachannel/internal/utils"
)

type authRepository struct{}

func NewAuthRepository() AuthRepository {
	return &authRepository{}
}

func scanUser(row interface{ Scan(dest ...interface{}) error }) (*User, error) {
	var u User
	var lastName, phone sql.NullString
	var avatar sql.NullString
	var lastLogin sql.NullTime
	var deletedAt utils.NullableTime
	var tenantID int

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

func scanRole(rows interface{ Scan(dest ...interface{}) error }) (Role, error) {
	var r Role
	err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.CreatedAt, &r.UpdatedAt)
	return r, err
}

func (r *authRepository) GetByEmail(ctx context.Context, q DBTX, tenantID int, email string) (*User, error) {
	query := `SELECT id, tenant_id, first_name, last_name, username, email, phone, password, avatar, last_login, deleted_at, created_at, updated_at FROM users WHERE email = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	return scanUser(q.QueryRowContext(ctx, query, email, tenantID))
}

func (r *authRepository) GetByUsername(ctx context.Context, q DBTX, tenantID int, username string) (*User, error) {
	query := `SELECT id, tenant_id, first_name, last_name, username, email, phone, password, avatar, last_login, deleted_at, created_at, updated_at FROM users WHERE username = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	return scanUser(q.QueryRowContext(ctx, query, username, tenantID))
}

func (r *authRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*User, error) {
	query := `SELECT id, tenant_id, first_name, last_name, username, email, phone, password, avatar, last_login, deleted_at, created_at, updated_at FROM users WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	return scanUser(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *authRepository) Create(ctx context.Context, q DBTX, user *User) (int, error) {
	query := `INSERT INTO users (tenant_id, first_name, last_name, username, email, phone, password, avatar, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	var avatarBytes []byte
	if user.Avatar != nil {
		avatarBytes = user.Avatar
	}
	var id int
	err := q.QueryRowContext(ctx, query,
		user.TenantID, user.FirstName, user.LastName, user.Username, user.Email, user.Phone,
		user.Password, avatarBytes, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *authRepository) GetRolesByUserID(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error) {
	query := `SELECT r.id, r.tenant_id, r.name, r.created_at, r.updated_at FROM roles r INNER JOIN role_user ru ON ru.role_id = r.id WHERE ru.user_id = $1 AND r.tenant_id = $2`
	rows, err := q.QueryContext(ctx, query, userID, tenantID)
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
