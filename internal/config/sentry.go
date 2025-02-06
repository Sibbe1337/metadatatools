package config

// SentryConfig holds Sentry error tracking configuration
type SentryConfig struct {
	DSN              string  `env:"SENTRY_DSN"`
	Environment      string  `env:"SENTRY_ENVIRONMENT" envDefault:"development"`
	Debug            bool    `env:"SENTRY_DEBUG" envDefault:"false"`
	SampleRate       float64 `env:"SENTRY_SAMPLE_RATE" envDefault:"1.0"`
	TracesSampleRate float64 `env:"SENTRY_TRACES_SAMPLE_RATE" envDefault:"0.2"`
}
