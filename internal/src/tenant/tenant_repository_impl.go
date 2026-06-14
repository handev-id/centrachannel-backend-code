package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type tenantRepository struct{}

func NewTenantRepository() TenantRepository {
	return &tenantRepository{}
}

func (r *tenantRepository) Create(ctx context.Context, q DBTX, tenant *Tenant) (int, error) {
	query := `INSERT INTO tenants (name, domain, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query,
		tenant.Name, tenant.Domain, true, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *tenantRepository) CreateRole(ctx context.Context, q DBTX, tenantID int, name string) (int, error) {
	query := `INSERT INTO roles (tenant_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query, tenantID, name, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *tenantRepository) CreateUser(ctx context.Context, q DBTX, user *User) (int, error) {
	query := `INSERT INTO users (tenant_id, first_name, last_name, username, email, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query,
		user.TenantID, user.FirstName, user.LastName, user.Username, user.Email, user.Password, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *tenantRepository) AttachRole(ctx context.Context, q DBTX, userID, roleID int) error {
	query := `INSERT INTO role_user (user_id, role_id, created_at, updated_at) VALUES ($1, $2, $3, $4)`
	_, err := q.ExecContext(ctx, query, userID, roleID, time.Now(), time.Now())
	return err
}

func (r *tenantRepository) List(ctx context.Context, q DBTX) ([]Tenant, error) {
	query := `SELECT id, name, domain, logo, address, phone, email, is_active, settings, created_at, updated_at FROM tenants ORDER BY created_at DESC`
	rows, err := q.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		var logo, settings sql.NullString
		var address, phone, email sql.NullString

		err := rows.Scan(&t.ID, &t.Name, &t.Domain, &logo, &address, &phone, &email, &t.IsActive, &settings, &t.CreatedAt, &t.UpdatedAt)
		if err != nil {
			return nil, err
		}

		if logo.Valid {
			t.Logo = json.RawMessage(logo.String)
		}
		if address.Valid {
			t.Address = &address.String
		}
		if phone.Valid {
			t.Phone = &phone.String
		}
		if email.Valid {
			t.Email = &email.String
		}
		if settings.Valid {
			t.Settings = json.RawMessage(settings.String)
		}

		tenants = append(tenants, t)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tenants, nil
}
