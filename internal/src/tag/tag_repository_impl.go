package tag

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type tagRepository struct{}

func NewTagRepository() TagRepository {
	return &tagRepository{}
}

func (r *tagRepository) List(ctx context.Context, q DBTX, tenantID int) ([]Tag, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, tenant_id, name, color, created_at, updated_at FROM tags WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []Tag
	for rows.Next() {
		var t Tag
		var color sql.NullString
		if err := rows.Scan(&t.ID, &t.TenantID, &t.Name, &color, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if color.Valid { t.Color = &color.String }
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (r *tagRepository) GetByID(ctx context.Context, q DBTX, tenantID int, id int) (*Tag, error) {
	var t Tag
	var color sql.NullString
	err := q.QueryRowContext(ctx, `SELECT id, tenant_id, name, color, created_at, updated_at FROM tags WHERE id = $1 AND tenant_id = $2`, id, tenantID).Scan(&t.ID, &t.TenantID, &t.Name, &color, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("tag not found")
	}
	if err != nil {
		return nil, err
	}
	if color.Valid { t.Color = &color.String }
	return &t, nil
}

func (r *tagRepository) Create(ctx context.Context, q DBTX, tag *Tag) (int, error) {
	query := `INSERT INTO tags (tenant_id, name, color, created_at, updated_at) VALUES ($1,$2,$3,$4,$5) RETURNING id`
	var id int
	err := q.QueryRowContext(ctx, query, tag.TenantID, tag.Name, tag.Color, time.Now(), time.Now()).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *tagRepository) Update(ctx context.Context, q DBTX, tenantID int, id int, tag *Tag) error {
	result, err := q.ExecContext(ctx, `UPDATE tags SET name=$1, color=$2, updated_at=$3 WHERE id=$4 AND tenant_id=$5`, tag.Name, tag.Color, time.Now(), id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tag not found")
	}
	return nil
}

func (r *tagRepository) Delete(ctx context.Context, q DBTX, tenantID int, id int) error {
	result, err := q.ExecContext(ctx, `DELETE FROM tags WHERE id=$1 AND tenant_id=$2`, id, tenantID)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("tag not found")
	}
	return nil
}
