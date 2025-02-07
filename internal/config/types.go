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
