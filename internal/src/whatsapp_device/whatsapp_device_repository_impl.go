package whatsapp_device

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type whatsAppDeviceRepository struct{}

func NewWhatsAppDeviceRepository() WhatsAppDeviceRepository {
	return &whatsAppDeviceRepository{}
}

func scanDevice(row interface {
	Scan(dest ...interface{}) error
}) (*WhatsAppDevice, error) {
	var d WhatsAppDevice
	var tenantID uuid.UUID

	err := row.Scan(&d.ID, &tenantID, &d.Name, &d.CountryCode, &d.Phone, &d.WhatsappID, &d.Status, &d.CreatedAt, &d.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("device not found")
	}
	if err != nil {
		return nil, err
	}
	d.TenantID = tenantID
	return &d, nil
}

func (r *whatsAppDeviceRepository) List(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]WhatsAppDevice, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, name, country_code, phone, whatsapp_id, status, created_at, updated_at FROM whatsapp_devices WHERE tenant_id = $1 ORDER BY name`, tenantID)
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

func (r *whatsAppDeviceRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*WhatsAppDevice, error) {
	return scanDevice(q.QueryRowContext(ctx, `SELECT id, tenant_id, name, country_code, phone, whatsapp_id, status, created_at, updated_at FROM whatsapp_devices WHERE id = $1 AND tenant_id = $2`, id, tenantID))
}

func (r *whatsAppDeviceRepository) GetByWhatsappID(ctx context.Context, q DBTX, whatsappID string) (*WhatsAppDevice, error) {
	return scanDevice(q.QueryRowContext(ctx, `SELECT id, tenant_id, name, country_code, phone, whatsapp_id, status, created_at, updated_at FROM whatsapp_devices WHERE whatsapp_id = $1`, whatsappID))
}

func (r *whatsAppDeviceRepository) Create(ctx context.Context, q DBTX, device *WhatsAppDevice) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRowContext(ctx, `INSERT INTO whatsapp_devices (tenant_id, name, country_code, phone, whatsapp_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`, device.TenantID, device.Name, device.CountryCode, device.Phone, device.WhatsappID, device.Status, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *whatsAppDeviceRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, device *WhatsAppDevice) error {
	result, err := q.ExecContext(ctx, `UPDATE whatsapp_devices SET name=$1, country_code=$2, phone=$3, status=$4, updated_at=$5 WHERE id=$6 AND tenant_id=$7`,
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

func (r *whatsAppDeviceRepository) Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
	result, err := q.ExecContext(ctx, `DELETE FROM whatsapp_devices WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}
