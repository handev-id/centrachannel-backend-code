package observability

import "context"

type observabilityRepository struct{}

func NewObservabilityRepository() ObservabilityRepository { return &observabilityRepository{} }
func (r *observabilityRepository) IsActiveDomain(ctx context.Context, q DBTX, name string) (bool, error) {
	var exists bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM domains d JOIN tenants t ON t.id=d.tenant_id WHERE lower(d.name)=$1 AND d.is_active AND t.is_active)`, name).Scan(&exists)
	return exists, err
}
