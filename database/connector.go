package database

import (
    "database/sql"
    "centrachannel/config"
)

// DBConnector abstracts a database connection provider.
// It allows services/repositories to depend on an interface instead of a concrete implementation,
// which simplifies testing (mocking) and follows the Dependency Injection rule.
type DBConnector interface {
    // Connect creates a new *sql.DB based on the supplied configuration.
    Connect(cfg *config.DatabaseConfig) (*sql.DB, error)
    // Close releases the underlying connection resources.
    Close(db *sql.DB) error
}

// PostgresConnector is the concrete implementation that uses the PostgreSQL driver.
type PostgresConnector struct{}

// Connect satisfies DBConnector by delegating to NewConnection.
func (p *PostgresConnector) Connect(cfg *config.DatabaseConfig) (*sql.DB, error) {
    return NewConnection(cfg)
}

// Close satisfies DBConnector by delegating to the Close helper.
func (p *PostgresConnector) Close(db *sql.DB) error {
    return Close(db)
}
