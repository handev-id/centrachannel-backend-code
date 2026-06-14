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
	CreateContact(ctx context.Context, q DBTX, campaignID int, contactID int) error
	UpdateContactStatus(ctx context.Context, q DBTX, campaignID int, contactID int, status string, errorMsg *string) error
	CountPendingContacts(ctx context.Context, q DBTX, campaignID int) (int, error)
	GetPendingContacts(ctx context.Context, q DBTX, campaignID int, limit int) ([]CampaignContact, error)
}
