package contact

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type contactRepository struct{}

func NewContactRepository() ContactRepository {
	return &contactRepository{}
}

func scanContact(row interface{ Scan(dest ...interface{}) error }) (*Contact, error) {
	var c Contact
	var lastName, username, email, phone, country, bio, occupation, category, categoryDesc, gender, province, facebook, instagram, whatsapp, x, tiktok, institutionName sql.NullString
	var avatar sql.NullString
	var dateOfBirth sql.NullTime
	var mergedToID sql.NullInt64
	var deletedAt sql.NullTime
	var tenantID int

	err := row.Scan(&c.ID, &tenantID, &c.FirstName, &lastName, &username, &email, &phone, &avatar, &country, &bio, &occupation, &category, &categoryDesc, &gender, &dateOfBirth, &province, &facebook, &instagram, &whatsapp, &x, &tiktok, &c.Status, &institutionName, &mergedToID, &deletedAt, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contact not found")
	}
	if err != nil {
		return nil, err
	}
	c.TenantID = tenantID
	if lastName.Valid { c.LastName = &lastName.String }
	if username.Valid { c.Username = &username.String }
	if email.Valid { c.Email = &email.String }
	if phone.Valid { c.Phone = &phone.String }
	if avatar.Valid { c.Avatar = []byte(avatar.String) }
	if country.Valid { c.Country = &country.String }
	if bio.Valid { c.Bio = &bio.String }
	if occupation.Valid { c.Occupation = &occupation.String }
	if category.Valid { c.Category = &category.String }
	if categoryDesc.Valid { c.CategoryDescription = &categoryDesc.String }
	if gender.Valid { c.Gender = &gender.String }
	if dateOfBirth.Valid { c.DateOfBirth = &dateOfBirth.Time }
	if province.Valid { c.ProvinceOfOrigin = &province.String }
	if facebook.Valid { c.Facebook = &facebook.String }
	if instagram.Valid { c.Instagram = &instagram.String }
	if whatsapp.Valid { c.Whatsapp = &whatsapp.String }
	if x.Valid { c.X = &x.String }
	if tiktok.Valid { c.Tiktok = &tiktok.String }
	if institutionName.Valid { c.InstitutionName = &institutionName.String }
	if mergedToID.Valid { id := int(mergedToID.Int64); c.MergedToID = &id }
	c.DeletedAt = deletedAt
	return &c, nil
}

func (r *contactRepository) List(ctx context.Context, q DBTX, tenantID int, limit, offset int, search, status string, channelID int) ([]*Contact, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("c.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, "c.deleted_at IS NULL")

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(c.first_name) LIKE LOWER($%d) OR LOWER(c.last_name) LIKE LOWER($%d) OR LOWER(c.email) LIKE LOWER($%d) OR LOWER(c.phone) LIKE LOWER($%d))", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}

	if channelID > 0 {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM profiles p WHERE p.contact_id = c.id AND p.channel_id = $%d)", argIdx))
		args = append(args, channelID)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM contacts c WHERE %s", where)
	var total int
	if err := q.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []*Contact{}, 0, nil
	}

	cols := `c.id, c.tenant_id, c.first_name, c.last_name, c.username, c.email, c.phone, c.avatar, c.country, c.bio, c.occupation, c.category, c.category_description, c.gender, c.date_of_birth, c.province_of_origin, c.facebook, c.instagram, c.whatsapp, c.x, c.tiktok, c.status, c.institution_name, c.merged_to_id, c.deleted_at, c.created_at, c.updated_at`
	dataQuery := fmt.Sprintf(`SELECT %s FROM contacts c WHERE %s ORDER BY c.created_at DESC LIMIT $%d OFFSET $%d`, cols, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := q.QueryContext(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var contacts []*Contact
	for rows.Next() {
		c, err := scanContact(rows)
		if err != nil {
			return nil, 0, err
		}
		contacts = append(contacts, c)
	}
	return contacts, total, rows.Err()
}

func (r *contactRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Contact, error) {
	query := `SELECT c.id, c.tenant_id, c.first_name, c.last_name, c.username, c.email, c.phone, c.avatar, c.country, c.bio, c.occupation, c.category, c.category_description, c.gender, c.date_of_birth, c.province_of_origin, c.facebook, c.instagram, c.whatsapp, c.x, c.tiktok, c.status, c.institution_name, c.merged_to_id, c.deleted_at, c.created_at, c.updated_at FROM contacts c WHERE c.id = $1 AND c.tenant_id = $2 AND c.deleted_at IS NULL`
	return scanContact(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *contactRepository) Create(ctx context.Context, q DBTX, contact *Contact) (int, error) {
	query := `INSERT INTO contacts (tenant_id, first_name, last_name, username, email, phone, avatar, country, bio, occupation, category, category_description, gender, date_of_birth, province_of_origin, facebook, instagram, whatsapp, x, tiktok, status, institution_name, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query,
		contact.TenantID, contact.FirstName, contact.LastName, contact.Username, contact.Email,
		contact.Phone, contact.Avatar, contact.Country, contact.Bio, contact.Occupation,
		contact.Category, contact.CategoryDescription, contact.Gender, contact.DateOfBirth,
		contact.ProvinceOfOrigin, contact.Facebook, contact.Instagram, contact.Whatsapp,
		contact.X, contact.Tiktok, contact.Status, contact.InstitutionName,
		time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *contactRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, contact *Contact) error {
	query := `UPDATE contacts SET first_name=$1, last_name=$2, username=$3, email=$4, phone=$5, avatar=COALESCE($6, avatar), country=$7, bio=$8, occupation=$9, category=$10, category_description=$11, gender=$12, date_of_birth=$13, province_of_origin=$14, facebook=$15, instagram=$16, whatsapp=$17, x=$18, tiktok=$19, status=$20, institution_name=$21, updated_at=$22 WHERE id=$23 AND tenant_id=$24 AND deleted_at IS NULL`
	result, err := q.ExecContext(ctx, query,
		contact.FirstName, contact.LastName, contact.Username, contact.Email,
		contact.Phone, contact.Avatar, contact.Country, contact.Bio, contact.Occupation,
		contact.Category, contact.CategoryDescription, contact.Gender, contact.DateOfBirth,
		contact.ProvinceOfOrigin, contact.Facebook, contact.Instagram, contact.Whatsapp,
		contact.X, contact.Tiktok, contact.Status, contact.InstitutionName,
		time.Now(), id, tenantID,
	)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("contact not found")
	}
	return nil
}

func (r *contactRepository) SoftDelete(ctx context.Context, q DBTX, tenantID int, id int) error {
	query := `UPDATE contacts SET deleted_at=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4 AND deleted_at IS NULL`
	result, err := q.ExecContext(ctx, query, time.Now(), time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("contact not found")
	}
	return nil
}
