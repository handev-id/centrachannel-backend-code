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
	var description, scheduledAt, sendingOption, stats sql.NullString
	var agentID, senderID, recipientListID, templateID sql.NullInt64

	err := row.Scan(&c.ID, &c.TenantID, &c.Name, &c.Type, &description, &c.MessageTemplate, &c.ChannelID, &c.Status, &sendingOption, &stats, &scheduledAt, &c.SentCount, &c.TotalCount, &agentID, &senderID, &recipientListID, &templateID, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if description.Valid { c.Description = &description.String }
	if scheduledAt.Valid {
		t, err := time.Parse(time.RFC3339, scheduledAt.String)
		if err == nil { c.ScheduledAt = &t }
	}
	if sendingOption.Valid { c.SendingOption = []byte(sendingOption.String) }
	if stats.Valid { c.Stats = []byte(stats.String) }
	if agentID.Valid { id := int(agentID.Int64); c.AgentID = &id }
	if senderID.Valid { id := int(senderID.Int64); c.SenderID = &id }
	if recipientListID.Valid { id := int(recipientListID.Int64); c.RecipientListID = &id }
	if templateID.Valid { id := int(templateID.Int64); c.TemplateID = &id }

	return &c, nil
}

func scanCampaignRecipient(row interface{ Scan(dest ...interface{}) error }) (*CampaignRecipient, error) {
	var r CampaignRecipient
	var failedReason sql.NullString
	var deliveryTime, openTime, clickTime sql.NullTime

	err := row.Scan(&r.ID, &r.CampaignID, &r.RecipientContactID, &r.Status, &failedReason, &deliveryTime, &openTime, &clickTime, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if failedReason.Valid { r.FailedReason = &failedReason.String }
	if deliveryTime.Valid { r.DeliveryTime = &deliveryTime.Time }
	if openTime.Valid { r.OpenTime = &openTime.Time }
	if clickTime.Valid { r.ClickTime = &clickTime.Time }

	return &r, nil
}

func scanTemplate(row interface{ Scan(dest ...interface{}) error }) (*CampaignTemplate, error) {
	var t CampaignTemplate
	var tp, templateType, category, lang, quality sql.NullString
	var accountID sql.NullInt64

	err := row.Scan(&t.ID, &t.TenantID, &t.Name, &tp, &templateType, &category, &lang, &t.Content, &t.Variables, &quality, &accountID, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if tp.Valid { t.Type = &tp.String }
	if templateType.Valid { t.TemplateType = &templateType.String }
	if category.Valid { t.Category = &category.String }
	if lang.Valid { t.Language = &lang.String }
	if quality.Valid { t.Quality = &quality.String }
	if accountID.Valid { id := int(accountID.Int64); t.AccountID = &id }

	return &t, nil
}

func scanRecipientList(row interface{ Scan(dest ...interface{}) error }) (*CampaignRecipientList, error) {
	var l CampaignRecipientList
	var status sql.NullString
	var channelID sql.NullInt64

	err := row.Scan(&l.ID, &l.TenantID, &l.Name, &l.Source, &status, &channelID, &l.CreatedAt, &l.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if status.Valid { l.Status = &status.String }
	if channelID.Valid { id := int(channelID.Int64); l.ChannelID = &id }

	return &l, nil
}

func scanRecipientContact(row interface{ Scan(dest ...interface{}) error }) (*CampaignRecipientContact, error) {
	var c CampaignRecipientContact
	var lastName, username, institution, email, phone sql.NullString
	var masterContactID sql.NullInt64

	err := row.Scan(&c.ID, &c.FirstName, &lastName, &username, &institution, &email, &phone, &c.CampaignRecipientListID, &masterContactID, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if lastName.Valid { c.LastName = &lastName.String }
	if username.Valid { c.Username = &username.String }
	if institution.Valid { c.Institution = &institution.String }
	if email.Valid { c.Email = &email.String }
	if phone.Valid { c.Phone = &phone.String }
	if masterContactID.Valid { id := int(masterContactID.Int64); c.MasterContactID = &id }

	return &c, nil
}

func (r *campaignRepository) List(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error) {
	countQuery := `SELECT COUNT(*) FROM campaigns WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%')`
	var total int
	err := q.QueryRowContext(ctx, countQuery, tenantID, search).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	dataQuery := `SELECT id, tenant_id, name, type, description, message_template, channel_id, status, sending_option::text, stats::text, scheduled_at::text, sent_count, total_count, agent_id, sender_id, recipient_list_id, template_id, created_by, created_at, updated_at FROM campaigns WHERE tenant_id = $1 AND ($2 = '' OR name ILIKE '%' || $2 || '%') ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	rows, err := q.QueryContext(ctx, dataQuery, tenantID, search, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var campaigns []Campaign
	for rows.Next() {
		c, err := scanCampaign(rows)
		if err != nil { return nil, 0, err }
		if c != nil { campaigns = append(campaigns, *c) }
	}
	return campaigns, total, rows.Err()
}

func (r *campaignRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error) {
	query := `SELECT id, tenant_id, name, type, description, message_template, channel_id, status, sending_option::text, stats::text, scheduled_at::text, sent_count, total_count, agent_id, sender_id, recipient_list_id, template_id, created_by, created_at, updated_at FROM campaigns WHERE id = $1 AND tenant_id = $2`
	return scanCampaign(q.QueryRowContext(ctx, query, id, tenantID))
}

func (r *campaignRepository) Create(ctx context.Context, q DBTX, campaign *Campaign) (int, error) {
	query := `INSERT INTO campaigns (tenant_id, name, type, description, message_template, channel_id, status, sending_option, scheduled_at, sent_count, total_count, agent_id, sender_id, recipient_list_id, template_id, created_by, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) RETURNING id`

	var scheduledAt interface{}
	if campaign.ScheduledAt != nil { scheduledAt = *campaign.ScheduledAt }

	var id int
	err := q.QueryRowContext(ctx, query,
		campaign.TenantID, campaign.Name, campaign.Type, campaign.Description, campaign.MessageTemplate,
		campaign.ChannelID, campaign.Status, campaign.SendingOption, scheduledAt,
		campaign.SentCount, campaign.TotalCount, campaign.AgentID, campaign.SenderID,
		campaign.RecipientListID, campaign.TemplateID, campaign.CreatedBy, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil { return 0, err }
	return id, nil
}

func (r *campaignRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error {
	var scheduledAt interface{}
	if campaign.ScheduledAt != nil { scheduledAt = *campaign.ScheduledAt }

	result, err := q.ExecContext(ctx, `UPDATE campaigns SET name=COALESCE($3,name), type=COALESCE($4,type), description=COALESCE($5,description), message_template=COALESCE($6,message_template), status=COALESCE($7,status), sending_option=COALESCE($8,sending_option), scheduled_at=COALESCE($9,scheduled_at), sent_count=COALESCE($10,sent_count), total_count=COALESCE($11,total_count), agent_id=COALESCE($12,agent_id), sender_id=COALESCE($13,sender_id), recipient_list_id=COALESCE($14,recipient_list_id), template_id=COALESCE($15,template_id), updated_at=$16 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, campaign.Name, campaign.Type, campaign.Description, campaign.MessageTemplate,
		campaign.Status, campaign.SendingOption, scheduledAt, campaign.SentCount, campaign.TotalCount,
		campaign.AgentID, campaign.SenderID, campaign.RecipientListID, campaign.TemplateID, time.Now(),
	)
	if err != nil { return err }

	rows, err := result.RowsAffected()
	if err != nil { return err }
	if rows == 0 { return fmt.Errorf("campaign not found") }
	return nil
}

func (r *campaignRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	result, err := q.ExecContext(ctx, `DELETE FROM campaigns WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil { return err }
	rows, _ := result.RowsAffected()
	if rows == 0 { return fmt.Errorf("campaign not found") }
	return nil
}

func (r *campaignRepository) CreateRecipient(ctx context.Context, q DBTX, recipient *CampaignRecipient) (int, error) {
	query := `INSERT INTO campaign_recipients (campaign_id, recipient_contact_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query, recipient.CampaignID, recipient.RecipientContactID, recipient.Status, time.Now(), time.Now()).Scan(&id)
	if err != nil { return 0, err }
	return id, nil
}

func (r *campaignRepository) UpdateRecipientStatus(ctx context.Context, q DBTX, id int, status string, failedReason *string, deliveryTime *sql.NullTime) error {
	query := `UPDATE campaign_recipients SET status=$2, failed_reason=$3, delivery_time=COALESCE($4,delivery_time), updated_at=$5 WHERE id=$1`
	_, err := q.ExecContext(ctx, query, id, status, failedReason, deliveryTime, time.Now())
	return err
}

func (r *campaignRepository) CountPendingRecipients(ctx context.Context, q DBTX, campaignID int) (int, error) {
	var count int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM campaign_recipients WHERE campaign_id=$1 AND status='pending'`, campaignID).Scan(&count)
	return count, err
}

func (r *campaignRepository) GetPendingRecipients(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignRecipient, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, campaign_id, recipient_contact_id, status, failed_reason, delivery_time, open_time, click_time, created_at, updated_at FROM campaign_recipients WHERE campaign_id=$1 AND status='pending' LIMIT $2`, campaignID, limit)
	if err != nil { return nil, err }
	defer rows.Close()

	var recipients []CampaignRecipient
	for rows.Next() {
		r, err := scanCampaignRecipient(rows)
		if err != nil { return nil, err }
		if r != nil { recipients = append(recipients, *r) }
	}
	return recipients, rows.Err()
}

func (r *campaignRepository) ListTemplates(ctx context.Context, q DBTX, tenantID int) ([]CampaignTemplate, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, name, type, template_type, category, language, content, variables, quality, account_id, created_at, updated_at FROM campaign_templates WHERE tenant_id=$1 ORDER BY name`, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()

	var templates []CampaignTemplate
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil { return nil, err }
		if t != nil { templates = append(templates, *t) }
	}
	return templates, rows.Err()
}

func (r *campaignRepository) GetTemplateByID(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignTemplate, error) {
	return scanTemplate(q.QueryRowContext(ctx, `SELECT id, tenant_id, name, type, template_type, category, language, content, variables, quality, account_id, created_at, updated_at FROM campaign_templates WHERE id=$1 AND tenant_id=$2`, id, tenantID))
}

func (r *campaignRepository) CreateTemplate(ctx context.Context, q DBTX, template *CampaignTemplate) (int, error) {
	query := `INSERT INTO campaign_templates (tenant_id, name, type, template_type, category, language, content, variables, quality, account_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query,
		template.TenantID, template.Name, template.Type, template.TemplateType, template.Category,
		template.Language, template.Content, template.Variables, template.Quality, template.AccountID,
		time.Now(), time.Now(),
	).Scan(&id)
	if err != nil { return 0, err }
	return id, nil
}

func (r *campaignRepository) UpdateTemplate(ctx context.Context, q DBTX, tenantID int, id int, template *CampaignTemplate) error {
	result, err := q.ExecContext(ctx, `UPDATE campaign_templates SET name=COALESCE($3,name), type=COALESCE($4,type), template_type=COALESCE($5,template_type), category=COALESCE($6,category), language=COALESCE($7,language), content=COALESCE($8,content), variables=COALESCE($9,variables), quality=COALESCE($10,quality), account_id=COALESCE($11,account_id), updated_at=$12 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, template.Name, template.Type, template.TemplateType, template.Category,
		template.Language, template.Content, template.Variables, template.Quality, template.AccountID, time.Now(),
	)
	if err != nil { return err }
	rows, _ := result.RowsAffected()
	if rows == 0 { return fmt.Errorf("template not found") }
	return nil
}

func (r *campaignRepository) DeleteTemplate(ctx context.Context, q DBTX, tenantID int, id int) error {
	result, err := q.ExecContext(ctx, `DELETE FROM campaign_templates WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil { return err }
	rows, _ := result.RowsAffected()
	if rows == 0 { return fmt.Errorf("template not found") }
	return nil
}

func (r *campaignRepository) ListRecipientLists(ctx context.Context, q DBTX, tenantID int) ([]CampaignRecipientList, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, name, source, status, channel_id, created_at, updated_at FROM campaign_recipient_lists WHERE tenant_id=$1 ORDER BY name`, tenantID)
	if err != nil { return nil, err }
	defer rows.Close()

	var lists []CampaignRecipientList
	for rows.Next() {
		l, err := scanRecipientList(rows)
		if err != nil { return nil, err }
		if l != nil { lists = append(lists, *l) }
	}
	return lists, rows.Err()
}

func (r *campaignRepository) GetRecipientListByID(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignRecipientList, error) {
	return scanRecipientList(q.QueryRowContext(ctx, `SELECT id, tenant_id, name, source, status, channel_id, created_at, updated_at FROM campaign_recipient_lists WHERE id=$1 AND tenant_id=$2`, id, tenantID))
}

func (r *campaignRepository) CreateRecipientList(ctx context.Context, q DBTX, list *CampaignRecipientList) (int, error) {
	query := `INSERT INTO campaign_recipient_lists (tenant_id, name, source, status, channel_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query, list.TenantID, list.Name, list.Source, list.Status, list.ChannelID, time.Now(), time.Now()).Scan(&id)
	if err != nil { return 0, err }
	return id, nil
}

func (r *campaignRepository) UpdateRecipientList(ctx context.Context, q DBTX, tenantID int, id int, list *CampaignRecipientList) error {
	result, err := q.ExecContext(ctx, `UPDATE campaign_recipient_lists SET name=COALESCE($3,name), source=COALESCE($4,source), status=COALESCE($5,status), channel_id=COALESCE($6,channel_id), updated_at=$7 WHERE id=$1 AND tenant_id=$2`,
		id, tenantID, list.Name, list.Source, list.Status, list.ChannelID, time.Now(),
	)
	if err != nil { return err }
	rows, _ := result.RowsAffected()
	if rows == 0 { return fmt.Errorf("recipient list not found") }
	return nil
}

func (r *campaignRepository) DeleteRecipientList(ctx context.Context, q DBTX, tenantID int, id int) error {
	result, err := q.ExecContext(ctx, `DELETE FROM campaign_recipient_lists WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil { return err }
	rows, _ := result.RowsAffected()
	if rows == 0 { return fmt.Errorf("recipient list not found") }
	return nil
}

func (r *campaignRepository) ListRecipientContacts(ctx context.Context, q DBTX, listID int) ([]CampaignRecipientContact, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, first_name, last_name, username, institution, email, phone, campaign_recipient_list_id, master_contact_id, created_at, updated_at FROM campaign_recipient_contacts WHERE campaign_recipient_list_id=$1 ORDER BY first_name`, listID)
	if err != nil { return nil, err }
	defer rows.Close()

	var contacts []CampaignRecipientContact
	for rows.Next() {
		c, err := scanRecipientContact(rows)
		if err != nil { return nil, err }
		if c != nil { contacts = append(contacts, *c) }
	}
	return contacts, rows.Err()
}

func (r *campaignRepository) CreateRecipientContact(ctx context.Context, q DBTX, contact *CampaignRecipientContact) (int, error) {
	query := `INSERT INTO campaign_recipient_contacts (first_name, last_name, username, institution, email, phone, campaign_recipient_list_id, master_contact_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query,
		contact.FirstName, contact.LastName, contact.Username, contact.Institution, contact.Email,
		contact.Phone, contact.CampaignRecipientListID, contact.MasterContactID, time.Now(), time.Now(),
	).Scan(&id)
	if err != nil { return 0, err }
	return id, nil
}

func (r *campaignRepository) DeleteRecipientContact(ctx context.Context, q DBTX, id int) error {
	result, err := q.ExecContext(ctx, `DELETE FROM campaign_recipient_contacts WHERE id=$1`, id)
	if err != nil { return err }
	rows, _ := result.RowsAffected()
	if rows == 0 { return fmt.Errorf("recipient contact not found") }
	return nil
}
