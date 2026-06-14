package campaign

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type campaignRepository struct{}

func NewCampaignRepository() CampaignRepository {
	return &campaignRepository{}
}

func scanCampaign(row interface{ Scan(dest ...interface{}) error }) (*Campaign, error) {
	var c Campaign
	var description, scheduledAt sql.NullString

	err := row.Scan(&c.ID, &c.TenantID, &c.Name, &description, &c.MessageTemplate, &c.ChannelID, &c.Status, &scheduledAt, &c.SentCount, &c.TotalCount, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if description.Valid {
		c.Description = &description.String
	}
	if scheduledAt.Valid {
		t, err := time.Parse(time.RFC3339, scheduledAt.String)
		if err == nil {
			c.ScheduledAt = &t
		}
	}

	return &c, nil
}

func scanCampaignContact(row interface{ Scan(dest ...interface{}) error }) (*CampaignContact, error) {
	var cc CampaignContact
	var sentAt sql.NullTime
	var errMsg sql.NullString

	err := row.Scan(&cc.ID, &cc.CampaignID, &cc.ContactID, &cc.Status, &sentAt, &errMsg, &cc.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if sentAt.Valid {
		cc.SentAt = &sentAt.Time
	}
	if errMsg.Valid {
		cc.ErrorMessage = &errMsg.String
	}

	return &cc, nil
}

func (r *campaignRepository) List(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error) {
	countQuery := `SELECT COUNT(*) FROM campaigns WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%')`
	var total int
	err := q.QueryRowContext(ctx, countQuery, tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery := `SELECT id, tenant_id, name, description, message_template, channel_id, status, scheduled_at::text, sent_count, total_count, created_by, created_at, updated_at FROM campaigns WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%') ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := q.QueryContext(ctx, dataQuery, tenantID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil {
			return nil, 0, err
		}
		if c != nil {
			campaigns = append(campaigns, *c)
		}
	}

	return campaigns, total, rows.Err()
}

func (r *campaignRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
	query := `SELECT id, tenant_id, name, description, message_template, channel_id, status, scheduled_at::text, sent_count, total_count, created_by, created_at, updated_at FROM campaigns WHERE id = $1 AND tenant_id = $2`
	return scanCampaign(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *campaignRepository) Create(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
	query := `INSERT INTO campaigns (tenant_id, name, description, message_template, channel_id, status, scheduled_at, sent_count, total_count, created_by, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12) RETURNING id`

	var scheduledAt interface{}
	if campaign.ScheduledAt != nil {
		scheduledAt = *campaign.ScheduledAt
	}

	var id int
	err := q.QueryRowContext(ctx, query,
		campaign.TenantID, campaign.Name, campaign.Description, campaign.MessageTemplate,
		campaign.ChannelID, campaign.Status, scheduledAt, campaign.SentCount, campaign.TotalCount,
		campaign.CreatedBy, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *campaignRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error {
	query := `UPDATE campaigns SET name = COALESCE($3, name), description = COALESCE($4, description), message_template = COALESCE($5, message_template), status = COALESCE($6, status), scheduled_at = COALESCE($7, scheduled_at), updated_at = $8 WHERE id = $1 AND tenant_id = $2`

	var scheduledAt interface{}
	if campaign.ScheduledAt != nil {
		scheduledAt = *campaign.ScheduledAt
	}

	result, err := q.ExecContext(ctx, query,
		id, tenantID, campaign.Name, campaign.Description, campaign.MessageTemplate,
		campaign.Status, scheduledAt, time.Now(),
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("campaign not found")
	}

	return nil
}

func (r *campaignRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	query := `DELETE FROM campaigns WHERE id = $1 AND tenant_id = $2`
	result, err := q.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("campaign not found")
	}
	return nil
}

func (r *campaignRepository) CreateContact(ctx context.Context, q DBTX, campaignID int, contactID int) error {
	query := `INSERT INTO campaign_contacts (campaign_id, contact_id, status, created_at) VALUES ($1, $2, 'pending', $3)`
	_, err := q.ExecContext(ctx, query, campaignID, contactID, time.Now())
	return err
}

func (r *campaignRepository) UpdateContactStatus(ctx context.Context, q DBTX, campaignID int, contactID int, status string, errorMsg *string) error {
	query := `UPDATE campaign_contacts SET status = $3, sent_at = CASE WHEN $3 = 'sent' THEN $4 ELSE sent_at END, error_message = $5 WHERE campaign_id = $1 AND contact_id = $2`
	_, err := q.ExecContext(ctx, query, campaignID, contactID, status, time.Now(), errorMsg)
	return err
}

func (r *campaignRepository) CountPendingContacts(ctx context.Context, q DBTX, campaignID int) (int, error) {
	query := `SELECT COUNT(*) FROM campaign_contacts WHERE campaign_id = $1 AND status = 'pending'`
	var count int
	err := q.QueryRowContext(ctx, query, campaignID).Scan(&count)
	return count, err
}

func (r *campaignRepository) GetPendingContacts(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignContact, error) {
	query := `SELECT id, campaign_id, contact_id, status, sent_at, error_message, created_at FROM campaign_contacts WHERE campaign_id = $1 AND status = 'pending' LIMIT $2`
	rows, err := q.QueryContext(ctx, query, campaignID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []CampaignContact
	for rows.Next() {
		cc, err := scanCampaignContact(rows)
		if err != nil {
			return nil, err
		}
		if cc != nil {
			contacts = append(contacts, *cc)
		}
	}
	return contacts, rows.Err()
}
