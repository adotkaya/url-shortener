package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration
// In Go, we use structs to group related data together
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	App      AppConfig
}

// ServerConfig holds HTTP server settings
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig holds PostgreSQL connection settings
type DatabaseConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
	CacheTTL time.Duration
}

// AppConfig holds application-specific settings
type AppConfig struct {
	Environment        string
	LogLevel           string
	ShortCodeLength    int
	RateLimitEnabled   bool
	RateLimitPerMinute int
	EnableAnalytics    bool
	EnableMetrics      bool
}

// Load reads configuration from environment variables
// This is a common pattern in Go - using environment variables for configuration
// makes your app portable across different environments (dev, staging, prod)
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  parseDuration("SERVER_READ_TIMEOUT", "10s"),
			WriteTimeout: parseDuration("SERVER_WRITE_TIMEOUT", "10s"),
			IdleTimeout:  parseDuration("SERVER_IDLE_TIMEOUT", "120s"),
		},
		Database: DatabaseConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "urlshortener"),
			Password:        getEnv("DB_PASSWORD", "dev_password_123"),
			DBName:          getEnv("DB_NAME", "urlshortener"),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    parseInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    parseInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: parseDuration("DB_CONN_MAX_LIFETIME", "5m"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       parseInt("REDIS_DB", 0),
			CacheTTL: parseDuration("REDIS_CACHE_TTL", "1h"),
		},
		App: AppConfig{
			Environment:        getEnv("APP_ENV", "development"),
			LogLevel:           getEnv("LOG_LEVEL", "info"),
			ShortCodeLength:    parseInt("SHORT_CODE_LENGTH", 6),
			RateLimitEnabled:   parseBool("RATE_LIMIT_ENABLED", true),
			RateLimitPerMinute: parseInt("RATE_LIMIT_REQUESTS_PER_MINUTE", 100),
			EnableAnalytics:    parseBool("ENABLE_ANALYTICS", true),
			EnableMetrics:      parseBool("ENABLE_METRICS", true),
		},
	}

	return cfg, nil
}

// DatabaseDSN returns the PostgreSQL connection string
// DSN = Data Source Name, a standard format for database connections
func (c *DatabaseConfig) DatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode,
	)
}

// RedisAddr returns the Redis address in host:port format
func (c *RedisConfig) RedisAddr() string {
	return fmt.Sprintf("%s:%s", c.Host, c.Port)
}

// Helper functions to parse environment variables with defaults
// These demonstrate error handling and type conversion in Go

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func parseBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return defaultValue
}

func parseDuration(key string, defaultValue string) time.Duration {
	value := getEnv(key, defaultValue)
	duration, err := time.ParseDuration(value)
	if err != nil {
		// If parsing fails, parse the default value
		duration, _ = time.ParseDuration(defaultValue)
	}
	return duration
}
