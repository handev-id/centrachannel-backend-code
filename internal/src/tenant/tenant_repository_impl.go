package tenant

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type tenantRepository struct{}

func NewTenantRepository() TenantRepository {
	return &tenantRepository{}
}

func (r *tenantRepository) Create(ctx context.Context, q DBTX, tenant *Tenant) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRowContext(ctx, `INSERT INTO tenants (name, domain, is_active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		tenant.Name, tenant.Domain, true, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *tenantRepository) CreateRole(ctx context.Context, q DBTX, tenantID uuid.UUID, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRowContext(ctx, `INSERT INTO roles (tenant_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id`, tenantID, name, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *tenantRepository) CreateUser(ctx context.Context, q DBTX, user *User) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRowContext(ctx, `INSERT INTO users (tenant_id, first_name, last_name, username, email, password, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		user.TenantID, user.FirstName, user.LastName, user.Username, user.Email, user.Password, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *tenantRepository) AttachRole(ctx context.Context, q DBTX, tenantID, userID, roleID uuid.UUID) error {
	_, err := q.ExecContext(ctx, `INSERT INTO role_user (tenant_id, user_id, role_id, created_at, updated_at) VALUES ($1, $2, $3, $4, $5)`, tenantID, userID, roleID, time.Now(), time.Now())
	return err
}

func (r *tenantRepository) GetByID(ctx context.Context, q DBTX, id uuid.UUID) (*Tenant, error) {
	return scanTenant(q.QueryRowContext(ctx, `SELECT id, name, domain, logo, address, phone, email, is_active, settings, created_at, updated_at FROM tenants WHERE id = $1`, id))
}

func (r *tenantRepository) GetByMetaPageID(ctx context.Context, q DBTX, pageID string) (*Tenant, error) {
	return scanTenant(q.QueryRowContext(ctx, `SELECT id, name, domain, logo, address, phone, email, is_active, settings, created_at, updated_at FROM tenants WHERE settings->'channel_configuration'->>'meta_page_id' = $1`, pageID))
}

func (r *tenantRepository) GetByMetaInstagramBusinessID(ctx context.Context, q DBTX, igID string) (*Tenant, error) {
	return scanTenant(q.QueryRowContext(ctx, `SELECT id, name, domain, logo, address, phone, email, is_active, settings, created_at, updated_at FROM tenants WHERE settings->'channel_configuration'->>'meta_instagram_business_id' = $1`, igID))
}

func scanTenant(row interface {
	Scan(dest ...interface{}) error
}) (*Tenant, error) {
	var t Tenant
	var logo, settings sql.NullString
	var address, phone, email sql.NullString

	err := row.Scan(&t.ID, &t.Name, &t.Domain, &logo, &address, &phone, &email, &t.IsActive, &settings, &t.CreatedAt, &t.UpdatedAt)
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

	return &t, nil
}

func (r *tenantRepository) Update(ctx context.Context, q DBTX, tenant *Tenant) error {
	_, err := q.ExecContext(ctx, `UPDATE tenants SET name=$1, domain=$2, logo=$3, address=$4, phone=$5, email=$6, is_active=$7, settings=$8, updated_at=$9 WHERE id=$10`,
		tenant.Name, tenant.Domain, tenant.Logo, tenant.Address, tenant.Phone, tenant.Email,
		tenant.IsActive, tenant.Settings, time.Now(), tenant.ID,
	)
	return err
}
