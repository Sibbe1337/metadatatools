package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// AppConfig holds all application configuration settings
type AppConfig struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	Redis    RedisConfig    `json:"redis"`
	Auth     AuthConfig     `json:"auth"`
	AI       AIConfig       `json:"ai"`
	Storage  StorageConfig  `json:"storage"`
	Session  SessionConfig  `json:"session"`
	Tracing  TracingConfig  `json:"tracing"`
	Jobs     JobsConfig     `json:"jobs"`
	Sentry   SentryConfig   `json:"sentry"`
	Queue    QueueConfig    `json:"queue"`
}

// ServerConfig holds server-related settings
type ServerConfig struct {
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	LogLevel    string `json:"log_level"`
	Address     string `json:"address"`
}

// DatabaseConfig holds database connection settings
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DBName   string `json:"dbname"`
	SSLMode  string `json:"sslmode"`
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Enabled  bool   `json:"enabled"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// GetAddress returns the formatted Redis address
func (c *RedisConfig) GetAddress() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// AuthConfig holds authentication settings
type AuthConfig struct {
	JWTSecret           string        `json:"jwt_secret"`
	AccessTokenTTL      time.Duration `json:"access_token_ttl"`
	RefreshTokenTTL     time.Duration `json:"refresh_token_ttl"`
	APIKeyLength        int           `json:"api_key_length"`
	PasswordMinLength   int           `json:"password_min_length"`
	PasswordHashCost    int           `json:"password_hash_cost"`
	MaxLoginAttempts    int           `json:"max_login_attempts"`
	LockoutDuration     time.Duration `json:"lockout_duration"`
	SessionTimeout      time.Duration `json:"session_timeout"`
	EnableTwoFactor     bool          `json:"enable_two_factor"`
	RequireStrongPasswd bool          `json:"require_strong_password"`
}

// AIConfig holds AI service settings
type AIConfig struct {
	Provider      string           `json:"provider"`
	APIKey        string           `json:"api_key"`
	ModelName     string           `json:"model_name"`
	ModelVersion  string           `json:"model_version"`
	Temperature   float64          `json:"temperature"`
	MaxTokens     int              `json:"max_tokens"`
	BatchSize     int              `json:"batch_size"`
	MinConfidence float64          `json:"min_confidence"`
	BaseURL       string           `json:"base_url"`
	Timeout       time.Duration    `json:"timeout"`
	Experiment    ExperimentConfig `json:"experiment"`
}

// ExperimentConfig holds A/B testing configuration
type ExperimentConfig struct {
	TrafficPercent float64 `json:"traffic_percent"`
	MinConfidence  float64 `json:"min_confidence"`
	EnableFallback bool    `json:"enable_fallback"`
}

// SessionConfig holds session management settings
type SessionConfig struct {
	CookieName         string        `json:"cookie_name"`
	CookieDomain       string        `json:"cookie_domain"`
	CookiePath         string        `json:"cookie_path"`
	CookieSecure       bool          `json:"cookie_secure"`
	CookieHTTPOnly     bool          `json:"cookie_http_only"`
	CookieSameSite     string        `json:"cookie_same_site"`
	SessionDuration    time.Duration `json:"session_duration"`
	CleanupInterval    time.Duration `json:"cleanup_interval"`
	MaxSessionsPerUser int           `json:"max_sessions_per_user"`
}

// TracingConfig holds tracing configuration settings
type TracingConfig struct {
	Enabled     bool    `json:"enabled"`
	ServiceName string  `json:"service_name"`
	Endpoint    string  `json:"endpoint"`
	SampleRate  float64 `json:"sample_rate"`
}

// JobsConfig holds background job settings
type JobsConfig struct {
	NumWorkers        int           `json:"num_workers"`
	MaxConcurrent     int           `json:"max_concurrent"`
	PollInterval      time.Duration `json:"poll_interval"`
	ShutdownWait      time.Duration `json:"shutdown_wait"`
	DefaultMaxRetries int           `json:"default_max_retries"`
	DefaultTTL        time.Duration `json:"default_ttl"`
	MaxPayloadSize    int64         `json:"max_payload_size"`
	QueuePrefix       string        `json:"queue_prefix"`
	RetryDelay        time.Duration `json:"retry_delay"`
	MaxRetryDelay     time.Duration `json:"max_retry_delay"`
	RetryMultiplier   float64       `json:"retry_multiplier"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	MaxJobAge         time.Duration `json:"max_job_age"`
}

// StorageConfig holds storage service settings
type StorageConfig struct {
	Provider         string        `json:"provider"`
	Region           string        `json:"region"`
	Bucket           string        `json:"bucket"`
	AccessKey        string        `json:"access_key"`
	SecretKey        string        `json:"secret_key"`
	Endpoint         string        `json:"endpoint"`
	UseSSL           bool          `json:"use_ssl"`
	UploadPartSize   int64         `json:"upload_part_size"`
	MaxUploadRetries int           `json:"max_upload_retries"`
	MaxFileSize      int64         `json:"max_file_size"`
	AllowedFileTypes []string      `json:"allowed_file_types"`
	UserQuota        int64         `json:"user_quota"`
	TotalQuota       int64         `json:"total_quota"`
	QuotaWarningPct  int           `json:"quota_warning_pct"`
	TempFileExpiry   time.Duration `json:"temp_file_expiry"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
	UploadBufferSize int64         `json:"upload_buffer_size"`
	DownloadTimeout  time.Duration `json:"download_timeout"`
	UploadTimeout    time.Duration `json:"upload_timeout"`
}

// SentryConfig holds Sentry error tracking configuration
type SentryConfig struct {
	DSN              string  `json:"dsn" env:"SENTRY_DSN"`
	Environment      string  `json:"environment" env:"SENTRY_ENVIRONMENT" envDefault:"development"`
	Debug            bool    `json:"debug" env:"SENTRY_DEBUG" envDefault:"false"`
	SampleRate       float64 `json:"sample_rate" env:"SENTRY_SAMPLE_RATE" envDefault:"1.0"`
	TracesSampleRate float64 `json:"traces_sample_rate" env:"SENTRY_TRACES_SAMPLE_RATE" envDefault:"0.2"`
}

// QueueConfig holds queue service configuration
type QueueConfig struct {
	Disabled           bool          `json:"disabled" env:"DISABLE_QUEUE" envDefault:"false"`
	ProjectID          string        `json:"project_id" env:"PUBSUB_PROJECT_ID,required"`
	HighPriorityTopic  string        `json:"high_priority_topic" env:"PUBSUB_HIGH_PRIORITY_TOPIC" envDefault:"high-priority"`
	LowPriorityTopic   string        `json:"low_priority_topic" env:"PUBSUB_LOW_PRIORITY_TOPIC" envDefault:"low-priority"`
	DeadLetterTopic    string        `json:"dead_letter_topic" env:"PUBSUB_DEAD_LETTER_TOPIC" envDefault:"dead-letter"`
	SubscriptionPrefix string        `json:"subscription_prefix" env:"PUBSUB_SUBSCRIPTION_PREFIX" envDefault:"sub"`
	MaxRetries         int           `json:"max_retries" env:"PUBSUB_MAX_RETRIES" envDefault:"3"`
	AckDeadline        time.Duration `json:"ack_deadline" env:"PUBSUB_ACK_DEADLINE" envDefault:"30s"`
	RetentionDuration  time.Duration `json:"retention_duration" env:"PUBSUB_RETENTION" envDefault:"168h"`
}

// Load loads configuration from environment variables
func Load() (*AppConfig, error) {
	cfg := &AppConfig{
		Server: ServerConfig{
			Port:        getEnvAsInt("SERVER_PORT", 8080),
			Environment: getEnvOrDefault("ENVIRONMENT", "development"),
			LogLevel:    getEnvOrDefault("LOG_LEVEL", "info"),
			Address:     getEnvOrDefault("SERVER_ADDRESS", ""),
		},
		Database: DatabaseConfig{
			Host:     getEnvOrDefault("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", ""),
			DBName:   getEnvOrDefault("DB_NAME", "metadatatool"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Enabled:  getEnvAsBool("REDIS_ENABLED", false),
			Host:     getEnvOrDefault("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnvOrDefault("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Auth: AuthConfig{
			JWTSecret:           getEnvOrDefault("JWT_SECRET", "your-secret-key"),
			AccessTokenTTL:      getEnvAsDuration("ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL:     getEnvAsDuration("REFRESH_TOKEN_TTL", 7*24*time.Hour),
			APIKeyLength:        getEnvAsInt("API_KEY_LENGTH", 32),
			PasswordMinLength:   getEnvAsInt("PASSWORD_MIN_LENGTH", 8),
			PasswordHashCost:    getEnvAsInt("PASSWORD_HASH_COST", 10),
			MaxLoginAttempts:    getEnvAsInt("MAX_LOGIN_ATTEMPTS", 5),
			LockoutDuration:     getEnvAsDuration("LOCKOUT_DURATION", 15*time.Minute),
			SessionTimeout:      getEnvAsDuration("SESSION_TIMEOUT", 24*time.Hour),
			EnableTwoFactor:     getEnvAsBool("ENABLE_TWO_FACTOR", false),
			RequireStrongPasswd: getEnvAsBool("REQUIRE_STRONG_PASSWORD", true),
		},
		AI: AIConfig{
			Provider:      getEnvOrDefault("AI_PROVIDER", "openai"),
			ModelName:     getEnvOrDefault("AI_MODEL_NAME", "gpt-4"),
			ModelVersion:  getEnvOrDefault("AI_MODEL_VERSION", "latest"),
			Temperature:   getEnvAsFloat("AI_TEMPERATURE", 0.7),
			MaxTokens:     getEnvAsInt("AI_MAX_TOKENS", 2048),
			BatchSize:     getEnvAsInt("AI_BATCH_SIZE", 10),
			MinConfidence: getEnvAsFloat("AI_MIN_CONFIDENCE", 0.85),
			APIKey:        getEnvOrDefault("AI_API_KEY", ""),
			BaseURL:       getEnvOrDefault("AI_BASE_URL", "https://api.openai.com/v1"),
			Timeout:       getEnvAsDuration("AI_TIMEOUT", 30*time.Second),
			Experiment: ExperimentConfig{
				TrafficPercent: getEnvAsFloat("AI_EXPERIMENT_TRAFFIC_PERCENT", 0.1),
				MinConfidence:  getEnvAsFloat("AI_MIN_CONFIDENCE_THRESHOLD", 0.8),
				EnableFallback: getEnvAsBool("AI_ENABLE_AUTO_FALLBACK", true),
			},
		},
		Session: SessionConfig{
			CookieName:         getEnvOrDefault("SESSION_COOKIE_NAME", "session"),
			CookieDomain:       getEnvOrDefault("SESSION_COOKIE_DOMAIN", ""),
			CookiePath:         getEnvOrDefault("SESSION_COOKIE_PATH", "/"),
			CookieSecure:       getEnvAsBool("SESSION_COOKIE_SECURE", true),
			CookieHTTPOnly:     getEnvAsBool("SESSION_COOKIE_HTTP_ONLY", true),
			CookieSameSite:     getEnvOrDefault("SESSION_COOKIE_SAME_SITE", "lax"),
			SessionDuration:    getEnvAsDuration("SESSION_DURATION", 24*time.Hour),
			CleanupInterval:    getEnvAsDuration("SESSION_CLEANUP_INTERVAL", time.Hour),
			MaxSessionsPerUser: getEnvAsInt("SESSION_MAX_PER_USER", 5),
		},
		Jobs: JobsConfig{
			NumWorkers:        getEnvAsInt("JOB_NUM_WORKERS", 5),
			MaxConcurrent:     getEnvAsInt("JOB_MAX_CONCURRENT", 10),
			PollInterval:      getEnvAsDuration("JOB_POLL_INTERVAL", time.Second),
			ShutdownWait:      getEnvAsDuration("JOB_SHUTDOWN_WAIT", 30*time.Second),
			DefaultMaxRetries: getEnvAsInt("JOB_DEFAULT_MAX_RETRIES", 3),
			DefaultTTL:        getEnvAsDuration("JOB_DEFAULT_TTL", 24*time.Hour),
			MaxPayloadSize:    getEnvAsInt64("JOB_MAX_PAYLOAD_SIZE", 1024*1024),
			QueuePrefix:       getEnvOrDefault("JOB_QUEUE_PREFIX", "jobs:"),
			RetryDelay:        getEnvAsDuration("JOB_RETRY_DELAY", 5*time.Second),
			MaxRetryDelay:     getEnvAsDuration("JOB_MAX_RETRY_DELAY", time.Hour),
			RetryMultiplier:   getEnvAsFloat("JOB_RETRY_MULTIPLIER", 2.0),
			CleanupInterval:   getEnvAsDuration("JOB_CLEANUP_INTERVAL", time.Hour),
			MaxJobAge:         getEnvAsDuration("JOB_MAX_AGE", 7*24*time.Hour),
		},
		Storage: StorageConfig{
			Provider:         getEnvOrDefault("STORAGE_PROVIDER", "s3"),
			Region:           getEnvOrDefault("STORAGE_REGION", "us-east-1"),
			Bucket:           getEnvOrDefault("STORAGE_BUCKET", "metadatatool"),
			AccessKey:        getEnvOrDefault("STORAGE_ACCESS_KEY", ""),
			SecretKey:        getEnvOrDefault("STORAGE_SECRET_KEY", ""),
			Endpoint:         getEnvOrDefault("STORAGE_ENDPOINT", ""),
			UseSSL:           getEnvAsBool("STORAGE_USE_SSL", true),
			UploadPartSize:   getEnvAsInt64("STORAGE_UPLOAD_PART_SIZE", 5*1024*1024),
			MaxUploadRetries: getEnvAsInt("STORAGE_MAX_UPLOAD_RETRIES", 3),
			MaxFileSize:      getEnvAsInt64("STORAGE_MAX_FILE_SIZE", 100*1024*1024),
			AllowedFileTypes: strings.Split(getEnvOrDefault("STORAGE_ALLOWED_FILE_TYPES", ".mp3,.wav,.flac"), ","),
			UserQuota:        getEnvAsInt64("STORAGE_USER_QUOTA", 1024*1024*1024),
			TotalQuota:       getEnvAsInt64("STORAGE_TOTAL_QUOTA", 1024*1024*1024*1024),
			QuotaWarningPct:  getEnvAsInt("STORAGE_QUOTA_WARNING_PCT", 90),
			TempFileExpiry:   getEnvAsDuration("STORAGE_TEMP_FILE_EXPIRY", 24*time.Hour),
			CleanupInterval:  getEnvAsDuration("STORAGE_CLEANUP_INTERVAL", time.Hour),
			UploadBufferSize: getEnvAsInt64("STORAGE_UPLOAD_BUFFER_SIZE", 5*1024*1024),
			DownloadTimeout:  getEnvAsDuration("STORAGE_DOWNLOAD_TIMEOUT", 5*time.Minute),
			UploadTimeout:    getEnvAsDuration("STORAGE_UPLOAD_TIMEOUT", 10*time.Minute),
		},
		Tracing: TracingConfig{
			Enabled:     getEnvAsBool("TRACING_ENABLED", true),
			ServiceName: getEnvOrDefault("TRACING_SERVICE_NAME", "metadatatool"),
			Endpoint:    getEnvOrDefault("TRACING_ENDPOINT", "localhost:4317"),
			SampleRate:  getEnvAsFloat("TRACING_SAMPLE_RATE", 0.1),
		},
		Sentry: SentryConfig{
			DSN:              getEnvOrDefault("SENTRY_DSN", ""),
			Environment:      getEnvOrDefault("SENTRY_ENVIRONMENT", "development"),
			Debug:            getEnvAsBool("SENTRY_DEBUG", false),
			SampleRate:       getEnvAsFloat("SENTRY_SAMPLE_RATE", 1.0),
			TracesSampleRate: getEnvAsFloat("SENTRY_TRACES_SAMPLE_RATE", 0.2),
		},
		Queue: QueueConfig{
			Disabled:           getEnvAsBool("DISABLE_QUEUE", false),
			ProjectID:          getEnvOrDefault("PUBSUB_PROJECT_ID", ""),
			HighPriorityTopic:  getEnvOrDefault("PUBSUB_HIGH_PRIORITY_TOPIC", "high-priority"),
			LowPriorityTopic:   getEnvOrDefault("PUBSUB_LOW_PRIORITY_TOPIC", "low-priority"),
			DeadLetterTopic:    getEnvOrDefault("PUBSUB_DEAD_LETTER_TOPIC", "dead-letter"),
			SubscriptionPrefix: getEnvOrDefault("PUBSUB_SUBSCRIPTION_PREFIX", "sub"),
			MaxRetries:         getEnvAsInt("PUBSUB_MAX_RETRIES", 3),
			AckDeadline:        getEnvAsDuration("PUBSUB_ACK_DEADLINE", 30*time.Second),
			RetentionDuration:  getEnvAsDuration("PUBSUB_RETENTION", 168*time.Hour),
		},
	}

	return cfg, nil
}

// Helper functions to get environment variables with defaults
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
