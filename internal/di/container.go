package di

import (
    "context"
    "database/sql"
    "fmt"

    "github.com/redis/go-redis/v9"

    "centrachannel/config"
    "centrachannel/database"
    "centrachannel/internal/utils/logger"
    "centrachannel/internal/ws"
)

type Container struct {
    Config *config.Config
    DB     *sql.DB
    Redis  *redis.Client
    Logger *logger.Logger
    Hub    *ws.Hub
}

func New() (*Container, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, err
    }

    connector := &database.PostgresConnector{}
    db, err := connector.Connect(&cfg.Database)
    if err != nil {
        return nil, err
    }

    rdb := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
    })

    if err := rdb.Ping(context.Background()).Err(); err != nil {
        return nil, fmt.Errorf("failed to connect to redis: %w", err)
    }

    l := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)
    hub := ws.NewHub()

    return &Container{Config: cfg, DB: db, Redis: rdb, Logger: l, Hub: hub}, nil
}

func (c *Container) StartHub() {
    go c.Hub.Run()
}

func (c *Container) Close() error {
    if c.Redis != nil {
        c.Redis.Close()
    }
    if c.DB != nil {
        connector := &database.PostgresConnector{}
        return connector.Close(c.DB)
    }
    return nil
}
