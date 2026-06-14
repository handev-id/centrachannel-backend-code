package channel

import "context"

type channelRepository struct{}

func NewChannelRepository() ChannelRepository {
	return &channelRepository{}
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
