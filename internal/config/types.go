// Package config provides configuration management for the application
package config

import "time"

// Config holds all configuration settings
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Auth     AuthConfig
	AI       AIConfig
	Storage  StorageConfig
	Jobs     JobConfig
}

// ServerConfig holds server-related settings
type ServerConfig struct {
	Port        int
	Environment string
	LogLevel    string
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Enabled  bool
	Host     string
	Port     int
	Password string
	DB       int
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	JWTSecret           string
	AccessTokenExpiry   time.Duration
	RefreshTokenExpiry  time.Duration
	APIKeyLength        int
	PasswordMinLength   int
	PasswordHashCost    int
	MaxLoginAttempts    int
	LockoutDuration     time.Duration
	SessionTimeout      time.Duration
	EnableTwoFactor     bool
	RequireStrongPasswd bool
}

// AIConfig holds AI service settings
type AIConfig struct {
	ModelName     string
	ModelVersion  string
	Temperature   float64
	MaxTokens     int
	BatchSize     int
	MinConfidence float64
	APIKey        string
	BaseURL       string
	Timeout       time.Duration
	Experiment    ExperimentConfig
}

// ExperimentConfig holds A/B testing configuration
type ExperimentConfig struct {
	TrafficPercent  float64
	MinConfidence   float64
	EnableFallback  bool
	BigQueryProject string
	BigQueryDataset string
}

// StorageConfig holds cloud storage configuration
type StorageConfig struct {
	// Provider settings
	Provider         string
	Region           string
	Bucket           string
	AccessKey        string
	SecretKey        string
	Endpoint         string
	UseSSL           bool
	UploadPartSize   int64
	MaxUploadRetries int

	// File restrictions
	MaxFileSize      int64
	AllowedFileTypes []string

	// Quota settings
	UserQuota       int64 // per user storage quota in bytes
	TotalQuota      int64 // total storage quota in bytes
	QuotaWarningPct int   // percentage at which to warn about quota

	// Cleanup settings
	TempFileExpiry  time.Duration // how long to keep temp files
	CleanupInterval time.Duration // how often to run cleanup

	// Performance settings
	UploadBufferSize int64         // size of upload buffer
	DownloadTimeout  time.Duration // timeout for download operations
	UploadTimeout    time.Duration // timeout for upload operations
}

// JobConfig holds job queue settings
type JobConfig struct {
	// Worker settings
	NumWorkers    int           `env:"JOB_NUM_WORKERS" envDefault:"5"`
	MaxConcurrent int           `env:"JOB_MAX_CONCURRENT" envDefault:"10"`
	PollInterval  time.Duration `env:"JOB_POLL_INTERVAL" envDefault:"1s"`
	ShutdownWait  time.Duration `env:"JOB_SHUTDOWN_WAIT" envDefault:"30s"`

	// Job settings
	DefaultMaxRetries int           `env:"JOB_DEFAULT_MAX_RETRIES" envDefault:"3"`
	DefaultTTL        time.Duration `env:"JOB_DEFAULT_TTL" envDefault:"24h"`
	MaxPayloadSize    int64         `env:"JOB_MAX_PAYLOAD_SIZE" envDefault:"1048576"` // 1MB

	// Queue settings
	QueuePrefix     string        `env:"JOB_QUEUE_PREFIX" envDefault:"jobs:"`
	RetryDelay      time.Duration `env:"JOB_RETRY_DELAY" envDefault:"5s"`
	MaxRetryDelay   time.Duration `env:"JOB_MAX_RETRY_DELAY" envDefault:"1h"`
	RetryMultiplier float64       `env:"JOB_RETRY_MULTIPLIER" envDefault:"2.0"`

	// Cleanup settings
	CleanupInterval time.Duration `env:"JOB_CLEANUP_INTERVAL" envDefault:"1h"`
	MaxJobAge       time.Duration `env:"JOB_MAX_AGE" envDefault:"168h"` // 7 days
}

// DefaultStorageConfig returns a default storage configuration
func DefaultStorageConfig() *StorageConfig {
	return &StorageConfig{
		Provider:         "s3",
		Region:           "us-east-1",
		UseSSL:           true,
		UploadPartSize:   5 * 1024 * 1024, // 5MB
		MaxUploadRetries: 3,

		MaxFileSize:      100 * 1024 * 1024, // 100MB
		AllowedFileTypes: []string{".mp3", ".wav", ".flac", ".m4a", ".aac"},

		UserQuota:       5 * 1024 * 1024 * 1024,   // 5GB
		TotalQuota:      100 * 1024 * 1024 * 1024, // 100GB
		QuotaWarningPct: 90,

		TempFileExpiry:  24 * time.Hour,
		CleanupInterval: 1 * time.Hour,

		UploadBufferSize: 1 * 1024 * 1024, // 1MB
		DownloadTimeout:  5 * time.Minute,
		UploadTimeout:    10 * time.Minute,
	}
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:        8080,
			Environment: "development",
			LogLevel:    "info",
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			DBName:   "metadatatool",
			SSLMode:  "disable",
		},
		Redis: RedisConfig{
			Enabled:  true,
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
		Auth: AuthConfig{
			AccessTokenExpiry:   15 * time.Minute,
			RefreshTokenExpiry:  7 * 24 * time.Hour,
			APIKeyLength:        32,
			PasswordMinLength:   8,
			PasswordHashCost:    10,
			MaxLoginAttempts:    5,
			LockoutDuration:     15 * time.Minute,
			SessionTimeout:      24 * time.Hour,
			EnableTwoFactor:     false,
			RequireStrongPasswd: true,
		},
		AI: AIConfig{
			ModelName:     "gpt-4",
			Temperature:   0.7,
			MaxTokens:     2048,
			BatchSize:     10,
			MinConfidence: 0.85,
			Timeout:       30 * time.Second,
		},
		Storage: *DefaultStorageConfig(),
		Jobs: JobConfig{
			NumWorkers:        5,
			MaxConcurrent:     10,
			PollInterval:      time.Second,
			ShutdownWait:      30 * time.Second,
			DefaultMaxRetries: 3,
			DefaultTTL:        24 * time.Hour,
			MaxPayloadSize:    1024 * 1024, // 1MB
			QueuePrefix:       "jobs:",
			RetryDelay:        5 * time.Second,
			MaxRetryDelay:     time.Hour,
			RetryMultiplier:   2.0,
			CleanupInterval:   time.Hour,
			MaxJobAge:         7 * 24 * time.Hour,
		},
	}
}
