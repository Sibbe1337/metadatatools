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
	Port        string
	Environment string
	LogLevel    string
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Enabled  bool
	Host     string
	Port     string
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
	Provider         string
	Region           string
	Bucket           string
	AccessKey        string
	SecretKey        string
	Endpoint         string
	UseSSL           bool
	UploadPartSize   int64
	MaxUploadRetries int
}
