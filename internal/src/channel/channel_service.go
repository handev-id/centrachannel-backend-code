package channel

import (
	"context"
	"database/sql"
)

type ChannelService interface {
	List(ctx context.Context) ([]Channel, error)
}

type channelService struct {
	repo ChannelRepository
	db   *sql.DB
}

func NewChannelService(repo ChannelRepository, db *sql.DB) ChannelService {
	return &channelService{repo: repo, db: db}
}

func (s *channelService) List(ctx context.Context) ([]Channel, error) {
	return s.repo.List(ctx, s.db)
}
