package config

import (
	"github.com/go-playground/validator/v10"

	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port int
	Env  string

	// Database
	Database DatabaseConfig

	// Redis
	Redis RedisConfig

	// JWT
	JWTSecret string
	JWTExpiry time.Duration

	// Email
	Email EmailConfig

	// Storage
	StorageURL       string
	StoragePath      string
	MaxFileSize      int64
	StorageSecretKey string
	LogLevel      string
	LogFormat     string

	// Evolution API
	EvolutionAPIURL   string
	EvolutionAPIKey   string
	WebhookBaseURL    string

	// Meta Webhook
	MetaWebhookSecret string
}

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

type EmailConfig struct {
	Host     string
	Port     int
	User     string
	Password string
}

func Load() (*Config, error) {
	// Load .env file if it exists (optional)
	_ = godotenv.Load()

	cfg := &Config{
		Port: getEnvInt("PORT", 3000),
		Env:  getEnv("ENV", "development"),
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnvInt("DB_PORT", 5432),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "centrachannel_db"),
			SSLMode:         getEnv("DB_SSL_MODE", "disable"),
			MaxOpenConns:    getEnvInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getEnvInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWTSecret:          getEnv("JWT_SECRET", "your-secret-key-change-this-in-production"),
		JWTExpiry:          getDurationEnv("JWT_EXPIRY", 24*time.Hour),
		Email: EmailConfig{
			Host:     getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:     getEnvInt("SMTP_PORT", 587),
			User:     getEnv("SMTP_USER", ""),
			Password: getEnv("SMTP_PASSWORD", ""),
		},
        StorageURL:       getEnv("STORAGE_URL", "https://storage.solodevs.my.id"),
        StoragePath:      getEnv("STORAGE_PATH", "./storage"),
        MaxFileSize:      getEnvInt64("MAX_FILE_SIZE", 10485760),
        StorageSecretKey: getEnv("STORAGE_SECRET_KEY", "solodevkeysss"),
        LogLevel:    getEnv("LOG_LEVEL", "info"),
        LogFormat:   getEnv("LOG_FORMAT", "json"),

        EvolutionAPIURL:  getEnv("EVOLUTION_API_URL", ""),
        EvolutionAPIKey:  getEnv("EVOLUTION_API_KEY", ""),
        WebhookBaseURL:   getEnv("WEBHOOK_BASE_URL", ""),
        MetaWebhookSecret: getEnv("META_WEBHOOK_SECRET", ""),

    }

    // Validate required fields using go-playground/validator
    v := validator.New()
    if err := v.Struct(cfg); err != nil {
        return nil, fmt.Errorf("configuration validation error: %w", err)
    }

    return cfg, nil
}

// Helper functions
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	valStr := getEnv(key, "")
	if val, err := strconv.ParseInt(valStr, 10, 64); err == nil {
		return val
	}
	return defaultVal
}

func getDurationEnv(key string, defaultVal time.Duration) time.Duration {
	valStr := getEnv(key, "")
	if val, err := time.ParseDuration(valStr); err == nil {
		return val
	}
	return defaultVal
}

func (cfg *Config) GetDatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
}
