package channel

import (
	"context"
	"fmt"
)

type channelRepository struct{}

func NewChannelRepository() ChannelRepository {
	return &channelRepository{}
}

func (r *channelRepository) GetByID(ctx context.Context, q DBTX, id int) (*Channel, error) {
	query := `SELECT id, name, type, logo, created_at, updated_at FROM channels WHERE id = $1`
	var c Channel
	err := q.QueryRowContext(ctx, query, id).Scan(&c.ID, &c.Name, &c.Type, &c.Logo, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}
	return &c, nil
}

func (r *channelRepository) GetByType(ctx context.Context, q DBTX, channelType string) (*Channel, error) {
	query := `SELECT id, name, type, logo, created_at, updated_at FROM channels WHERE type = $1`
	var c Channel
	err := q.QueryRowContext(ctx, query, channelType).Scan(&c.ID, &c.Name, &c.Type, &c.Logo, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("channel not found: %w", err)
	}
	return &c, nil
}

func (r *channelRepository) List(ctx context.Context, q DBTX) ([]Channel, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, name, type, logo, created_at, updated_at FROM channels ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []Channel
	for rows.Next() {
		var c Channel
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.Logo, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		channels = append(channels, c)
	}
	return channels, rows.Err()
}
