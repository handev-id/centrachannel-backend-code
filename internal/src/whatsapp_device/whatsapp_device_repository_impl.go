package whatsapp_device

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type whatsAppDeviceRepository struct{}

func NewWhatsAppDeviceRepository() WhatsAppDeviceRepository {
	return &whatsAppDeviceRepository{}
}

func scanDevice(row interface{ Scan(dest ...interface{}) error }) (*WhatsAppDevice, error) {
	var d WhatsAppDevice
	var deletedAt sql.NullTime
	var tenantID int

	err := row.Scan(&d.ID, &tenantID, &d.Name, &d.CountryCode, &d.Phone, &d.WhatsappID, &d.Status, &deletedAt, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	if err != nil {
		return nil, err
	}
	d.TenantID = tenantID
	d.DeletedAt = deletedAt
	return &d, nil
}

func (r *whatsAppDeviceRepository) List(ctx context.Context, q DBTX, tenantID int) ([]WhatsAppDevice, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, name, country_code, phone, whatsapp_id, status, deleted_at, created_at, updated_at FROM whatsapp_devices WHERE tenant_id = $1 AND deleted_at IS NULL ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var devices []WhatsAppDevice
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		devices = append(devices, *d)
	}
	return devices, rows.Err()
}

func (r *whatsAppDeviceRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*WhatsAppDevice, error) {
	query := `SELECT id, tenant_id, name, country_code, phone, whatsapp_id, status, deleted_at, created_at, updated_at FROM whatsapp_devices WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	return scanDevice(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *whatsAppDeviceRepository) Create(ctx context.Context, q DBTX, device *WhatsAppDevice) (int, error) {
	query := `INSERT INTO whatsapp_devices (tenant_id, name, country_code, phone, whatsapp_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query, device.TenantID, device.Name, device.CountryCode, device.Phone, device.WhatsappID, device.Status, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *whatsAppDeviceRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, device *WhatsAppDevice) error {
	result, err := q.ExecContext(ctx, `UPDATE whatsapp_devices SET name=$1, country_code=$2, phone=$3, status=$4, updated_at=$5 WHERE id=$6 AND tenant_id=$7 AND deleted_at IS NULL`,
		device.Name, device.CountryCode, device.Phone, device.Status, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}

func (r *whatsAppDeviceRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	result, err := q.ExecContext(ctx, `UPDATE whatsapp_devices SET deleted_at=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4 AND deleted_at IS NULL`, time.Now(), time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}
