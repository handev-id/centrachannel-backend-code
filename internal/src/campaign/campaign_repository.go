package campaign

import (
	"context"
	"database/sql"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type CampaignRepository interface {
	List(ctx context.Context, q DBTX, tenantID int, limit int, offset int, search string) ([]Campaign, int, error)
	GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Campaign, error)
	Create(ctx context.Context, q DBTX, campaign *Campaign) (int, error)
	Update(ctx context.Context, q DBTX, tenantID int, id int, campaign *Campaign) error
	Delete(ctx context.Context, q DBTX, tenantID int, id int) error
	CreateRecipient(ctx context.Context, q DBTX, recipient *CampaignRecipient) (int, error)
	UpdateRecipientStatus(ctx context.Context, q DBTX, id int, status string, failedReason *string, deliveryTime *sql.NullTime) error
	CountPendingRecipients(ctx context.Context, q DBTX, campaignID int) (int, error)
	GetPendingRecipients(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignRecipient, error)
	ListTemplates(ctx context.Context, q DBTX, tenantID int) ([]CampaignTemplate, error)
	GetTemplateByID(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignTemplate, error)
	CreateTemplate(ctx context.Context, q DBTX, template *CampaignTemplate) (int, error)
	UpdateTemplate(ctx context.Context, q DBTX, tenantID int, id int, template *CampaignTemplate) error
	DeleteTemplate(ctx context.Context, q DBTX, tenantID int, id int) error
	ListRecipientLists(ctx context.Context, q DBTX, tenantID int) ([]CampaignRecipientList, error)
	GetRecipientListByID(ctx context.Context, q DBTX, tenantID int, id int) (*CampaignRecipientList, error)
	CreateRecipientList(ctx context.Context, q DBTX, list *CampaignRecipientList) (int, error)
	UpdateRecipientList(ctx context.Context, q DBTX, tenantID int, id int, list *CampaignRecipientList) error
	DeleteRecipientList(ctx context.Context, q DBTX, tenantID int, id int) error
	ListRecipientContacts(ctx context.Context, q DBTX, listID int) ([]CampaignRecipientContact, error)
	CreateRecipientContact(ctx context.Context, q DBTX, contact *CampaignRecipientContact) (int, error)
	DeleteRecipientContact(ctx context.Context, q DBTX, id int) error
}
