package config

import (
	"os"
	"strconv"
	"time"
)

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Server config
	cfg.Server = ServerConfig{
		Port:        getEnvAsInt("SERVER_PORT", 8080),
		Environment: getEnvOrDefault("ENVIRONMENT", "development"),
		LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
	}

	// Database config
	cfg.Database = DatabaseConfig{
		Host:     getEnvOrDefault("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnvOrDefault("DB_USER", "postgres"),
		Password: getEnvOrDefault("DB_PASSWORD", ""),
		DBName:   getEnvOrDefault("DB_NAME", "metadatatool"),
		SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
	}

	// Redis config
	cfg.Redis = RedisConfig{
		Enabled:  getEnvAsBool("REDIS_ENABLED", false),
		Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
		Port:     getEnvAsInt("REDIS_PORT", 6379),
		Password: getEnvOrDefault("REDIS_PASSWORD", ""),
		DB:       getEnvAsInt("REDIS_DB", 0),
	}

	// Auth config
	cfg.Auth = AuthConfig{
		JWTSecret:           getEnvOrDefault("JWT_SECRET", "your-secret-key"),
		AccessTokenExpiry:   getEnvAsDuration("ACCESS_TOKEN_EXPIRY", 15*time.Minute),
		RefreshTokenExpiry:  getEnvAsDuration("REFRESH_TOKEN_EXPIRY", 7*24*time.Hour),
		APIKeyLength:        getEnvAsInt("API_KEY_LENGTH", 32),
		PasswordMinLength:   getEnvAsInt("PASSWORD_MIN_LENGTH", 8),
		PasswordHashCost:    getEnvAsInt("PASSWORD_HASH_COST", 10),
		MaxLoginAttempts:    getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
		LockoutDuration:     getEnvAsDuration("LOCKOUT_DURATION", 15*time.Minute),
		SessionTimeout:      getEnvAsDuration("SESSION_TIMEOUT", 24*time.Hour),
		EnableTwoFactor:     getEnvAsBool("ENABLE_TWO_FACTOR", false),
		RequireStrongPasswd: getEnvAsBool("REQUIRE_STRONG_PASSWORD", true),
	}

	// AI config
	cfg.AI = AIConfig{
		ModelName:     getEnvOrDefault("AI_MODEL_NAME", "gpt-4"),
		ModelVersion:  getEnvOrDefault("AI_MODEL_VERSION", "latest"),
		Temperature:   getEnvAsFloat("AI_TEMPERATURE", 0.7),
		MaxTokens:     getEnvAsInt("AI_MAX_TOKENS", 2048),
		BatchSize:     getEnvAsInt("AI_BATCH_SIZE", 10),
		MinConfidence: getEnvAsFloat("AI_MIN_CONFIDENCE", 0.85),
		APIKey:        getEnvOrDefault("AI_API_KEY", ""),
		BaseURL:       getEnvOrDefault("AI_BASE_URL", "https://api.openai.com/v1"),
		Timeout:       getEnvAsDuration("AI_TIMEOUT", 30*time.Second),
	}

	// Storage config
	cfg.Storage = StorageConfig{
		Provider:         getEnvOrDefault("STORAGE_PROVIDER", "s3"),
		Region:           getEnvOrDefault("STORAGE_REGION", "us-east-1"),
		Bucket:           getEnvOrDefault("STORAGE_BUCKET", "metadatatool"),
		AccessKey:        getEnvOrDefault("STORAGE_ACCESS_KEY", ""),
		SecretKey:        getEnvOrDefault("STORAGE_SECRET_KEY", ""),
		Endpoint:         getEnvOrDefault("STORAGE_ENDPOINT", ""),
		UseSSL:           getEnvAsBool("STORAGE_USE_SSL", true),
		UploadPartSize:   getEnvAsInt64("STORAGE_UPLOAD_PART_SIZE", 5*1024*1024), // 5MB
		MaxUploadRetries: getEnvAsInt("STORAGE_MAX_UPLOAD_RETRIES", 3),
	}

	return cfg, nil
}

// Helper functions for environment variable parsing

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
