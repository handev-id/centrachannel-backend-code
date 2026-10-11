package campaign

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

type DBTX interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type CampaignRepository interface {
	List(ctx context.Context, q DBTX, tenantID uuid.UUID, limit int, offset int, search string) ([]*Campaign, int, error)
	GetByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*Campaign, error)
	Create(ctx context.Context, q DBTX, campaign *Campaign) (uuid.UUID, error)
	Update(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, campaign *Campaign) error
	UpdateStatus(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, status string, sentCount *int) error
	Delete(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	CreateRecipient(ctx context.Context, q DBTX, recipient *CampaignRecipient) (uuid.UUID, error)
	UpdateRecipientStatus(ctx context.Context, q DBTX, id uuid.UUID, status string, failedReason *string, deliveryTime *sql.NullTime) error
	CountPendingRecipients(ctx context.Context, q DBTX, campaignID uuid.UUID) (int, error)
	GetPendingRecipients(ctx context.Context, q DBTX, campaignID uuid.UUID, limit int) ([]CampaignRecipient, error)
	GetPendingRecipientsWithPhone(ctx context.Context, q DBTX, campaignID uuid.UUID, limit int) ([]CampaignRecipientWithPhone, error)
	ListTemplates(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]CampaignTemplate, error)
	GetTemplateByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*CampaignTemplate, error)
	CreateTemplate(ctx context.Context, q DBTX, template *CampaignTemplate) (uuid.UUID, error)
	UpdateTemplate(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, template *CampaignTemplate) error
	DeleteTemplate(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	ListRecipientLists(ctx context.Context, q DBTX, tenantID uuid.UUID) ([]CampaignRecipientList, error)
	GetRecipientListByID(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) (*CampaignRecipientList, error)
	CreateRecipientList(ctx context.Context, q DBTX, list *CampaignRecipientList) (uuid.UUID, error)
	UpdateRecipientList(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID, list *CampaignRecipientList) error
	DeleteRecipientList(ctx context.Context, q DBTX, tenantID uuid.UUID, id uuid.UUID) error
	ListRecipientContacts(ctx context.Context, q DBTX, tenantID uuid.UUID, listID uuid.UUID) ([]CampaignRecipientContact, error)
	CreateRecipientContact(ctx context.Context, q DBTX, contact *CampaignRecipientContact) (uuid.UUID, error)
	DeleteRecipientContact(ctx context.Context, q DBTX, tenantID uuid.UUID, listID uuid.UUID, id uuid.UUID) error
}
