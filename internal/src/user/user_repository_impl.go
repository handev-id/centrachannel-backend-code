package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"centrachannel/internal/utils"
	"github.com/lib/pq"
)

type userRepository struct{}

func NewUserRepository() UserRepository {
	return &userRepository{}
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
		return nil, fmt.Errorf("user not found")
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

func (r *userRepository) List(ctx context.Context, q DBTX, tenantID int, limit, offset int, search string, roleID *int) ([]*User, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("u.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, "u.deleted_at IS NULL")

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(u.first_name) LIKE LOWER($%d) OR LOWER(u.last_name) LIKE LOWER($%d) OR LOWER(u.email) LIKE LOWER($%d) OR LOWER(u.phone) LIKE LOWER($%d))", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if roleID != nil {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM role_user ru WHERE ru.tenant_id = u.tenant_id AND ru.user_id = u.id AND ru.role_id = $%d)", argIdx))
		args = append(args, *roleID)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users u WHERE %s", whereClause)
	var total int
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	if total == 0 {
		return []*User{}, 0, nil
	}

	dataQuery := fmt.Sprintf(`SELECT u.id, u.tenant_id, u.first_name, u.last_name, u.username, u.email, u.phone, u.password, u.avatar, u.last_login, u.deleted_at, u.created_at, u.updated_at FROM users u WHERE %s ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d`, whereClause, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := q.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

func (r *userRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*User, error) {
	query := `SELECT id, tenant_id, first_name, last_name, username, email, phone, password, avatar, last_login, deleted_at, created_at, updated_at FROM users WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	return scanUser(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *userRepository) Create(ctx context.Context, q DBTX, user *User) (int, error) {
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

func (r *userRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, user *User) error {
	query := `UPDATE users SET first_name = $1, last_name = $2, username = $3, email = $4, phone = $5, password = COALESCE(NULLIF($6, ''), password), avatar = COALESCE($7, avatar), updated_at = $8 WHERE id = $9 AND tenant_id = $10 AND deleted_at IS NULL`
	var avatarBytes []byte
	if user.Avatar != nil {
		avatarBytes = user.Avatar
	}
	result, err := q.ExecContext(ctx, query, user.FirstName, user.LastName, user.Username, user.Email, user.Phone, user.Password, avatarBytes, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *userRepository) SoftDelete(ctx context.Context, q DBTX, tenantID int, id int) error {
	query := `UPDATE users SET deleted_at = $1, updated_at = $2 WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL`
	result, err := q.ExecContext(ctx, query, time.Now(), time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (r *userRepository) GetRolesByUserID(ctx context.Context, q DBTX, tenantID int, userID int) ([]Role, error) {
	query := `SELECT r.id, r.tenant_id, r.name, r.created_at, r.updated_at FROM roles r INNER JOIN role_user ru ON ru.role_id = r.id WHERE ru.user_id = $1 AND r.tenant_id = $2`
	rows, err := q.QueryContext(ctx, query, userID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []Role
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.ID, &r.TenantID, &r.Name, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, r)
	}
	return roles, rows.Err()
}

func (r *userRepository) GetRolesByUserIDs(ctx context.Context, q DBTX, tenantID int, userIDs []int) (map[int][]Role, error) {
	if len(userIDs) == 0 {
		return map[int][]Role{}, nil
	}

	query := `SELECT ru.user_id, r.id, r.tenant_id, r.name, r.created_at, r.updated_at FROM roles r INNER JOIN role_user ru ON ru.role_id = r.id WHERE ru.user_id = ANY($1) AND r.tenant_id = $2`
	rows, err := q.QueryContext(ctx, query, pq.Array(userIDs), tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[int][]Role, len(userIDs))
	for rows.Next() {
		var userID int
		var role Role
		if err := rows.Scan(&userID, &role.ID, &role.TenantID, &role.Name, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		result[userID] = append(result[userID], role)
	}
	return result, rows.Err()
}

func (r *userRepository) AttachRoles(ctx context.Context, q DBTX, tenantID int, userID int, roleIDs []int) error {
	if len(roleIDs) == 0 {
		return nil
	}

	query := `INSERT INTO role_user (tenant_id, user_id, role_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING`
	for _, roleID := range roleIDs {
		if _, err := q.ExecContext(ctx, query, tenantID, userID, roleID, time.Now(), time.Now()); err != nil {
			return err
		}
	}
	return nil
}

func (r *userRepository) SyncRoles(ctx context.Context, q DBTX, tenantID int, userID int, roleIDs []int) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM role_user WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID); err != nil {
		return err
	}
	return r.AttachRoles(ctx, q, tenantID, userID, roleIDs)
}
