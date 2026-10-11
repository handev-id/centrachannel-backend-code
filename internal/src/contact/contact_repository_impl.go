package contact

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"centrachannel/internal/utils"
)

type contactRepository struct{}

func NewContactRepository() ContactRepository {
	return &contactRepository{}
}

func scanContact(row interface {
	Scan(dest ...interface{}) error
}) (*Contact, error) {
	var c Contact
	var lastName, username, email, phone, country, bio, occupation, category, categoryDesc, gender, province, facebook, instagram, whatsapp, x, tiktok, institutionName sql.NullString
	var avatar sql.NullString
	var dateOfBirth sql.NullTime
	var mergedToID uuid.NullUUID
	var deletedAt utils.NullableTime
	var tenantID uuid.UUID

	err := row.Scan(&c.ID, &tenantID, &c.FirstName, &lastName, &username, &email, &phone, &avatar, &country, &bio, &occupation, &category, &categoryDesc, &gender, &dateOfBirth, &province, &facebook, &instagram, &whatsapp, &x, &tiktok, &c.Status, &institutionName, &mergedToID, &deletedAt, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("contact not found")
	}
	if err != nil {
		return nil, err
	}
	c.TenantID = tenantID
	if lastName.Valid {
		c.LastName = &lastName.String
	}
	if username.Valid {
		c.Username = &username.String
	}
	if email.Valid {
		c.Email = &email.String
	}
	if phone.Valid {
		c.Phone = &phone.String
	}
	if avatar.Valid {
		c.Avatar = []byte(avatar.String)
	}
	if country.Valid {
		c.Country = &country.String
	}
	if bio.Valid {
		c.Bio = &bio.String
	}
	if occupation.Valid {
		c.Occupation = &occupation.String
	}
	if category.Valid {
		c.Category = &category.String
	}
	if categoryDesc.Valid {
		c.CategoryDescription = &categoryDesc.String
	}
	if gender.Valid {
		c.Gender = &gender.String
	}
	if dateOfBirth.Valid {
		c.DateOfBirth = &dateOfBirth.Time
	}
	if province.Valid {
		c.ProvinceOfOrigin = &province.String
	}
	if facebook.Valid {
		c.Facebook = &facebook.String
	}
	if instagram.Valid {
		c.Instagram = &instagram.String
	}
	if whatsapp.Valid {
		c.Whatsapp = &whatsapp.String
	}
	if x.Valid {
		c.X = &x.String
	}
	if tiktok.Valid {
		c.Tiktok = &tiktok.String
	}
	if institutionName.Valid {
		c.InstitutionName = &institutionName.String
	}
	if mergedToID.Valid {
		id := mergedToID.UUID
		c.MergedToID = &id
	}
	c.DeletedAt = deletedAt
	return &c, nil
}

func (r *contactRepository) List(ctx context.Context, q DBTX, tenantID uuid.UUID, limit, offset int, f ListContactQuery) ([]*Contact, int, error) {
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("c.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	conditions = append(conditions, "c.deleted_at IS NULL")

	if f.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(LOWER(c.first_name) LIKE LOWER($%d) OR LOWER(c.last_name) LIKE LOWER($%d) OR LOWER(c.email) LIKE LOWER($%d) OR LOWER(c.phone) LIKE LOWER($%d))", argIdx, argIdx, argIdx, argIdx))
		args = append(args, "%"+f.Search+"%")
		argIdx++
	}

	if f.Status != "" {
		conditions = append(conditions, fmt.Sprintf("c.status = $%d", argIdx))
		args = append(args, f.Status)
		argIdx++
	}

	if f.ChannelID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM profiles p WHERE p.contact_id = c.id AND p.channel_id = $%d)", argIdx))
		args = append(args, f.ChannelID)
		argIdx++
	}

	if f.ChannelType != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM profiles p INNER JOIN channels ch ON ch.id = p.channel_id WHERE p.contact_id = c.id AND ch.type = $%d)", argIdx))
		args = append(args, f.ChannelType)
		argIdx++
	}

	if f.Category != "" {
		conditions = append(conditions, fmt.Sprintf("c.category = $%d", argIdx))
		args = append(args, f.Category)
		argIdx++
	}

	if f.Country != "" {
		conditions = append(conditions, fmt.Sprintf("c.country = $%d", argIdx))
		args = append(args, f.Country)
		argIdx++
	}

	if f.Province != "" {
		conditions = append(conditions, fmt.Sprintf("c.province_of_origin = $%d", argIdx))
		args = append(args, f.Province)
		argIdx++
	}

	if f.IsMerged == "true" {
		conditions = append(conditions, "c.merged_to_id IS NOT NULL")
	} else if f.IsMerged == "false" {
		conditions = append(conditions, "c.merged_to_id IS NULL")
	}

	if f.AgentID != uuid.Nil {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM conversations cv WHERE cv.tenant_id = c.tenant_id AND cv.agent_id = $%d AND EXISTS (SELECT 1 FROM profiles p WHERE p.id = cv.profile_id AND p.contact_id = c.id))", argIdx))
		args = append(args, f.AgentID)
		argIdx++
	}

	if f.HasConversation == "true" {
		conditions = append(conditions, "EXISTS (SELECT 1 FROM conversations cv INNER JOIN profiles p ON p.id = cv.profile_id WHERE p.contact_id = c.id AND cv.tenant_id = c.tenant_id)")
	} else if f.HasConversation == "false" {
		conditions = append(conditions, "NOT EXISTS (SELECT 1 FROM conversations cv INNER JOIN profiles p ON p.id = cv.profile_id WHERE p.contact_id = c.id AND cv.tenant_id = c.tenant_id)")
	}

	if f.LastActivityFrom != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM conversations cv INNER JOIN profiles p ON p.id = cv.profile_id WHERE p.contact_id = c.id AND cv.tenant_id = c.tenant_id AND cv.last_activity >= $%d)", argIdx))
		args = append(args, f.LastActivityFrom)
		argIdx++
	}

	if f.LastActivityTo != "" {
		conditions = append(conditions, fmt.Sprintf("EXISTS (SELECT 1 FROM conversations cv INNER JOIN profiles p ON p.id = cv.profile_id WHERE p.contact_id = c.id AND cv.tenant_id = c.tenant_id AND cv.last_activity <= $%d)", argIdx))
		args = append(args, f.LastActivityTo)
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

	orderBy := "c.created_at DESC"
	if f.SortBy != "" {
		switch f.SortBy {
		case "first_name", "last_name", "email", "phone", "created_at", "updated_at":
			orderBy = "c." + f.SortBy + " ASC"
		case "-first_name", "-last_name", "-email", "-phone", "-created_at", "-updated_at":
			orderBy = "c." + f.SortBy[1:] + " DESC"
		}
	}

	cols := `c.id, c.tenant_id, c.first_name, c.last_name, c.username, c.email, c.phone, c.avatar, c.country, c.bio, c.occupation, c.category, c.category_description, c.gender, c.date_of_birth, c.province_of_origin, c.facebook, c.instagram, c.whatsapp, c.x, c.tiktok, c.status, c.institution_name, c.merged_to_id, c.deleted_at, c.created_at, c.updated_at`
	dataQuery := fmt.Sprintf(`SELECT %s FROM contacts c WHERE %s ORDER BY %s LIMIT $%d OFFSET $%d`, cols, where, orderBy, argIdx, argIdx+1)
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

func (r *contactRepository) GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Contact, error) {
	query := `SELECT c.id, c.tenant_id, c.first_name, c.last_name, c.username, c.email, c.phone, c.avatar, c.country, c.bio, c.occupation, c.category, c.category_description, c.gender, c.date_of_birth, c.province_of_origin, c.facebook, c.instagram, c.whatsapp, c.x, c.tiktok, c.status, c.institution_name, c.merged_to_id, c.deleted_at, c.created_at, c.updated_at FROM contacts c WHERE c.id = $1 AND c.tenant_id = $2 AND c.deleted_at IS NULL`
	return scanContact(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *contactRepository) GetByPhone(ctx context.Context, q DBTX, tenantID uuid.UUID, phone string) (*Contact, error) {
	query := `SELECT c.id, c.tenant_id, c.first_name, c.last_name, c.username, c.email, c.phone, c.avatar, c.country, c.bio, c.occupation, c.category, c.category_description, c.gender, c.date_of_birth, c.province_of_origin, c.facebook, c.instagram, c.whatsapp, c.x, c.tiktok, c.status, c.institution_name, c.merged_to_id, c.deleted_at, c.created_at, c.updated_at FROM contacts c WHERE c.tenant_id = $1 AND c.phone = $2 AND c.deleted_at IS NULL`
	return scanContact(q.QueryRowContext(ctx, query, tenantID, phone))
}

func (r *contactRepository) Create(ctx context.Context, q DBTX, contact *Contact) (uuid.UUID, error) {
	query := `INSERT INTO contacts (tenant_id, first_name, last_name, username, email, phone, avatar, country, bio, occupation, category, category_description, gender, date_of_birth, province_of_origin, facebook, instagram, whatsapp, x, tiktok, status, institution_name, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24) RETURNING id`
	var id uuid.UUID
	err := q.QueryRowContext(ctx, query,
		contact.TenantID, contact.FirstName, contact.LastName, contact.Username, contact.Email,
		contact.Phone, contact.Avatar, contact.Country, contact.Bio, contact.Occupation,
		contact.Category, contact.CategoryDescription, contact.Gender, contact.DateOfBirth,
		contact.ProvinceOfOrigin, contact.Facebook, contact.Instagram, contact.Whatsapp,
		contact.X, contact.Tiktok, contact.Status, contact.InstitutionName,
		time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func (r *contactRepository) Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, contact *Contact) error {
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

func (r *contactRepository) SoftDelete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error {
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

func (r *contactRepository) ListConversations(ctx context.Context, q DBTX, tenantID, contactID uuid.UUID, limit, offset int) ([]ConversationBrief, int, error) {
	var total int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM conversations c
		INNER JOIN profiles p ON p.id = c.profile_id AND p.tenant_id = c.tenant_id AND p.deleted_at IS NULL
		WHERE p.contact_id = $1 AND c.tenant_id = $2
	`, contactID, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := q.QueryContext(ctx, `
		SELECT c.id, c.status, c.profile_id, c.agent_id, c.channel_id,
		       c.unread_count, c.last_message, c.last_activity, c.created_at
		FROM conversations c
		INNER JOIN profiles p ON p.id = c.profile_id AND p.tenant_id = c.tenant_id AND p.deleted_at IS NULL
		WHERE p.contact_id = $1 AND c.tenant_id = $2
		ORDER BY c.last_activity DESC NULLS LAST, c.id DESC
		LIMIT $3 OFFSET $4
	`, contactID, tenantID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var convs []ConversationBrief
	for rows.Next() {
		var conv ConversationBrief
		var agentID uuid.NullUUID
		var lastMessage sql.NullString
		var lastActivity sql.NullTime

		if err := rows.Scan(&conv.ID, &conv.Status, &conv.ProfileID, &agentID, &conv.ChannelID, &conv.UnreadCount, &lastMessage, &lastActivity, &conv.CreatedAt); err != nil {
			return nil, 0, err
		}
		if agentID.Valid {
			id := agentID.UUID
			conv.AgentID = &id
		}
		if lastMessage.Valid {
			conv.LastMessage = []byte(lastMessage.String)
		}
		if lastActivity.Valid {
			conv.LastActivity = &lastActivity.Time
		}

		convs = append(convs, conv)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return convs, total, nil
}

func (r *contactRepository) Merge(ctx context.Context, q DBTX, tenantID, sourceID, targetID uuid.UUID) error {
	if sourceID == targetID {
		return fmt.Errorf("cannot merge contact into itself")
	}
	// The service supplies a transaction. Lock both contacts in a stable order so
	// overlapping merges cannot change their ownership while we move profiles.
	rows, err := q.QueryContext(ctx, `SELECT id FROM contacts WHERE id IN ($1,$2) AND tenant_id = $3 AND deleted_at IS NULL ORDER BY id FOR UPDATE`, sourceID, targetID, tenantID)
	if err != nil {
		return err
	}
	count := 0
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if count != 2 {
		return fmt.Errorf("source or target contact not found")
	}
	// A single provenance column cannot represent nested merges. Require the
	// incoming merges to be undone before merging their target elsewhere.
	var incoming bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM contacts WHERE tenant_id = $1 AND merged_to_id = $2)`, tenantID, sourceID).Scan(&incoming); err != nil {
		return err
	}
	if incoming {
		return fmt.Errorf("unmerge source contact's merged contacts first")
	}
	if _, err := q.ExecContext(ctx, `UPDATE contacts SET merged_to_id = $1, deleted_at = $2, updated_at = $2 WHERE id = $3 AND tenant_id = $4 AND deleted_at IS NULL`, targetID, time.Now(), sourceID, tenantID); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `UPDATE profiles SET contact_id = $1, merged_from_contact_id = $2, updated_at = $4 WHERE contact_id = $2 AND tenant_id = $3 AND deleted_at IS NULL`, targetID, sourceID, tenantID, time.Now()); err != nil {
		return err
	}

	return nil
}

func (r *contactRepository) Unmerge(ctx context.Context, q DBTX, tenantID, id uuid.UUID) error {
	result, err := q.ExecContext(ctx, `UPDATE contacts SET merged_to_id = NULL, deleted_at = NULL, updated_at = $1 WHERE id = $2 AND tenant_id = $3 AND merged_to_id IS NOT NULL`, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("merged contact not found")
	}
	if _, err := q.ExecContext(ctx, `UPDATE profiles SET contact_id = $1, merged_from_contact_id = NULL, updated_at = $3 WHERE merged_from_contact_id = $1 AND tenant_id = $2`, id, tenantID, time.Now()); err != nil {
		return err
	}

	return nil
}
