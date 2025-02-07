package retry

import (
	"time"
)

// Function represents a function that can be retried
type Function func() error

// Option represents a retry option
type Option func(*Config)

// Config holds retry configuration
type Config struct {
	attempts uint
	delay    time.Duration
	maxDelay time.Duration
	onRetry  func(uint, error)
}

// Attempts sets the number of retry attempts
func Attempts(attempts uint) Option {
	return func(c *Config) {
		c.attempts = attempts
	}
}

// Delay sets the delay between retries
func Delay(delay time.Duration) Option {
	return func(c *Config) {
		c.delay = delay
	}
}

// MaxDelay sets the maximum delay between retries
func MaxDelay(maxDelay time.Duration) Option {
	return func(c *Config) {
		c.maxDelay = maxDelay
	}
}

// OnRetry sets the function to call on retry
func OnRetry(onRetry func(uint, error)) Option {
	return func(c *Config) {
		c.onRetry = onRetry
	}
}

// Do executes the function with retry logic
func Do(fn Function, opts ...Option) error {
	config := &Config{
		attempts: 1,
		delay:    1 * time.Second,
		maxDelay: 30 * time.Second,
	}

	for _, opt := range opts {
		opt(config)
	}

	var err error
	for attempt := uint(0); attempt < config.attempts; attempt++ {
		err = fn()
		if err == nil {
			return nil
		}

		if attempt+1 < config.attempts {
			if config.onRetry != nil {
				config.onRetry(attempt+1, err)
			}

			delay := config.delay * time.Duration(attempt+1)
			if delay > config.maxDelay {
				delay = config.maxDelay
			}

			time.Sleep(delay)
		}
	}

	return err
}
