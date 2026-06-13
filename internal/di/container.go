package di

import (
    "database/sql"
    "centrachannel/config"
    "centrachannel/database"
    "centrachannel/internal/utils/logger"
)

// Container provides the core shared components of the application.
// It is responsible for loading configuration, establishing the database
// connection, and creating the logger instance. All services and repositories
// can receive the needed dependencies from the container.

type Container struct {
    Config *config.Config
    DB     *sql.DB
    Logger *logger.Logger
}

// New creates a new Container, loading configuration, initializing the DB
// connection and the logger. Any error aborts the creation so the caller can
// fail fast.
func New() (*Container, error) {
    cfg, err := config.Load()
    if err != nil {
        return nil, err
    }

    // Use the concrete PostgresConnector defined in the database package.
    connector := &database.PostgresConnector{}
    db, err := connector.Connect(&cfg.Database)
    if err != nil {
        return nil, err
    }

    l := logger.NewLogger(cfg.LogLevel, cfg.LogFormat)

    return &Container{Config: cfg, DB: db, Logger: l}, nil
}

// Close releases the DB connection held by the container.
func (c *Container) Close() error {
    if c.DB != nil {
        connector := &database.PostgresConnector{}
        return connector.Close(c.DB)
    }
    return nil
}
