package profile

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type profileRepository struct{}

func NewProfileRepository() ProfileRepository {
	return &profileRepository{}
}

func scanProfile(row interface{ Scan(dest ...interface{}) error }) (*Profile, error) {
	var p Profile
	var username, displayName, linkedDevice sql.NullString
	var mergedFromID sql.NullInt64
	var deletedAt sql.NullTime

	err := row.Scan(&p.ID, &p.ExternalID, &username, &displayName, &p.IsMain, &linkedDevice, &mergedFromID, &p.ContactID, &p.ChannelID, &deletedAt, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("profile not found")
	}
	if err != nil {
		return nil, err
	}
	if username.Valid { p.Username = &username.String }
	if displayName.Valid { p.DisplayName = &displayName.String }
	if linkedDevice.Valid { p.LinkedDeviceWhatsappID = &linkedDevice.String }
	if mergedFromID.Valid { id := int(mergedFromID.Int64); p.MergedFromContactID = &id }
	p.DeletedAt = deletedAt
	return &p, nil
}

func (r *profileRepository) List(ctx context.Context, q DBTX, contactID, channelID int) ([]Profile, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, "p.deleted_at IS NULL")

	if contactID > 0 {
		conditions = append(conditions, fmt.Sprintf("p.contact_id = $%d", argIdx))
		args = append(args, contactID)
		argIdx++
	}
	if channelID > 0 {
		conditions = append(conditions, fmt.Sprintf("p.channel_id = $%d", argIdx))
		args = append(args, channelID)
		argIdx++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + joinStrings(conditions, " AND ")
	}

	query := fmt.Sprintf(`SELECT p.id, p.external_id, p.username, p.display_name, p.is_main, p.linked_device_whatsapp_id, p.merged_from_contact_id, p.contact_id, p.channel_id, p.deleted_at, p.created_at, p.updated_at FROM profiles p %s ORDER BY p.id`, where)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		pp, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *pp)
	}
	return profiles, rows.Err()
}

func (r *profileRepository) GetByID(ctx context.Context, q DBTX, id int) (*Profile, error) {
	query := `SELECT id, external_id, username, display_name, is_main, linked_device_whatsapp_id, merged_from_contact_id, contact_id, channel_id, deleted_at, created_at, updated_at FROM profiles WHERE id = $1 AND deleted_at IS NULL`
	return scanProfile(q.QueryRowContext(ctx, query, id))
}

func (r *profileRepository) Update(ctx context.Context, q DBTX, id int, profile *Profile) error {
	query := `UPDATE profiles SET is_main=$1, linked_device_whatsapp_id=$2, display_name=$3, updated_at=$4 WHERE id=$5 AND deleted_at IS NULL`
	result, err := q.ExecContext(ctx, query, profile.IsMain, profile.LinkedDeviceWhatsappID, profile.DisplayName, time.Now(), id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("profile not found")
	}
	return nil
}

func (r *profileRepository) GetByContactID(ctx context.Context, q DBTX, contactID int) ([]Profile, error) {
	query := `SELECT id, external_id, username, display_name, is_main, linked_device_whatsapp_id, merged_from_contact_id, contact_id, channel_id, deleted_at, created_at, updated_at FROM profiles WHERE contact_id = $1 AND deleted_at IS NULL ORDER BY id`
	rows, err := q.QueryContext(ctx, query, contactID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, *p)
	}
	return profiles, rows.Err()
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
