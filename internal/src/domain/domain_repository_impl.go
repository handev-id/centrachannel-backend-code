package domain

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type domainRepository struct{}

func NewDomainRepository() DomainRepository { return &domainRepository{} }

func (r *domainRepository) GetByTenantIDs(ctx context.Context, q DBTX, ids []uuid.UUID) (map[uuid.UUID][]Domain, error) {
	result := make(map[uuid.UUID][]Domain, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = id.String()
		result[id] = []Domain{}
	}
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, name, is_default, is_active, created_at, updated_at FROM domains WHERE tenant_id = ANY($1::uuid[]) ORDER BY is_default DESC, name, id`, pq.Array(values))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.IsDefault, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		result[d.TenantID] = append(result[d.TenantID], d)
	}
	return result, rows.Err()
}
func (r *domainRepository) List(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]Domain, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, name, is_default, is_active, created_at, updated_at FROM domains WHERE tenant_id=$1 ORDER BY is_default DESC, name, id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var domains []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.TenantID, &d.Name, &d.IsDefault, &d.IsActive, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		domains = append(domains, d)
	}
	return domains, rows.Err()
}
func (r *domainRepository) GetByID(ctx context.Context, q DBTX, tenantID, id uuid.UUID) (*Domain, error) {
	var d Domain
	err := q.QueryRowContext(ctx, `SELECT id, tenant_id, name, is_default, is_active, created_at, updated_at FROM domains WHERE tenant_id=$1 AND id=$2`, tenantID, id).Scan(&d.ID, &d.TenantID, &d.Name, &d.IsDefault, &d.IsActive, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}
func (r *domainRepository) Create(ctx context.Context, q DBTX, d *Domain) (uuid.UUID, error) {
	var tx *sql.Tx
	if d.IsDefault {
		if db, ok := q.(*sql.DB); ok {
			var err error
			tx, err = db.BeginTx(ctx, nil)
			if err != nil {
				return uuid.Nil, err
			}
			defer tx.Rollback()
			q = tx
		}
		var tenantID uuid.UUID
		if err := q.QueryRowContext(ctx, `SELECT id FROM tenants WHERE id=$1 FOR UPDATE`, d.TenantID).Scan(&tenantID); err != nil {
			return uuid.Nil, err
		}
		if _, err := q.ExecContext(ctx, `UPDATE domains SET is_default=FALSE,updated_at=NOW() WHERE tenant_id=$1 AND is_default`, d.TenantID); err != nil {
			return uuid.Nil, err
		}
	}
	var id uuid.UUID
	err := q.QueryRowContext(ctx, `INSERT INTO domains (tenant_id,name,is_default,is_active) VALUES ($1,$2,$3,$4) RETURNING id,created_at,updated_at`, d.TenantID, d.Name, d.IsDefault, d.IsActive).Scan(&id, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.Nil, ErrConflict
		}
		return uuid.Nil, err
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return uuid.Nil, err
		}
	}
	return id, nil
}
func (r *domainRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, d *Domain) error {
	var tx *sql.Tx
	if d.IsDefault {
		if db, ok := q.(*sql.DB); ok {
			var err error
			tx, err = db.BeginTx(ctx, nil)
			if err != nil {
				return err
			}
			defer tx.Rollback()
			q = tx
		}
		var lockedID uuid.UUID
		if err := q.QueryRowContext(ctx, `SELECT id FROM tenants WHERE id=$1 FOR UPDATE`, tenantID).Scan(&lockedID); err != nil {
			return err
		}
		if _, err := q.ExecContext(ctx, `UPDATE domains SET is_default=FALSE,updated_at=NOW() WHERE tenant_id=$1 AND is_default`, tenantID); err != nil {
			return err
		}
	}
	// The condition also protects a domain promoted by a concurrent request after the service read it.
	err := q.QueryRowContext(ctx, `UPDATE domains SET name=$1,is_default=$2,is_active=$3,updated_at=NOW() WHERE tenant_id=$4 AND id=$5 AND (NOT is_default OR $2) RETURNING updated_at`, d.Name, d.IsDefault, d.IsActive, tenantID, id).Scan(&d.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		if _, lookupErr := r.GetByID(ctx, q, tenantID, id); lookupErr != nil {
			return lookupErr
		}
		return ErrDefault
	}
	if err != nil {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return err
	}
	if tx != nil {
		return tx.Commit()
	}
	return nil
}
func (r *domainRepository) Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	var deletedID uuid.UUID
	err := q.QueryRowContext(ctx, `DELETE FROM domains WHERE tenant_id=$1 AND id=$2 AND NOT is_default RETURNING id`, tenantID, id).Scan(&deletedID)
	if errors.Is(err, sql.ErrNoRows) {
		if _, lookupErr := r.GetByID(ctx, q, tenantID, id); lookupErr != nil {
			return lookupErr
		}
		return ErrDefault
	}
	return err
}
