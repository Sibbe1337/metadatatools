package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Timer is a utility for timing operations and recording their duration
type Timer struct {
	observer prometheus.Observer
	start    time.Time
}

// NewTimer creates a new timer that will observe the duration using the given observer
func NewTimer(observer prometheus.Observer) *Timer {
	return &Timer{
		observer: observer,
		start:    time.Now(),
	}
}

// ObserveDuration records the duration since the timer was created
func (t *Timer) ObserveDuration() {
	t.observer.Observe(time.Since(t.start).Seconds())
}
