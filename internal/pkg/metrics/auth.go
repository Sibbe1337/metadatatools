package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Authentication metrics
	AuthAttempts = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "auth_attempts_total",
		Help: "Total number of authentication attempts",
	}, []string{"method", "status"}) // method: login, refresh, api_key; status: success, failure

	ActiveSessions = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "active_sessions_total",
		Help: "Total number of active sessions",
	})

	SessionOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "session_operations_total",
		Help: "Total number of session operations",
	}, []string{"operation", "status"}) // operation: create, delete, refresh; status: success, failure

	TokenOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "token_operations_total",
		Help: "Total number of token operations",
	}, []string{"operation", "status"}) // operation: generate, validate, refresh; status: success, failure

	APIKeyOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "api_key_operations_total",
		Help: "Total number of API key operations",
	}, []string{"operation", "status"}) // operation: generate, validate; status: success, failure

	AuthLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "auth_operation_duration_seconds",
		Help:    "Duration of authentication operations",
		Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
	}, []string{"operation"}) // operation: login, refresh, validate, session_create, etc.

	// Role and permission metrics
	PermissionChecks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "permission_checks_total",
		Help: "Total number of permission checks",
	}, []string{"permission", "status"}) // status: allowed, denied

	RoleChecks = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "role_checks_total",
		Help: "Total number of role checks",
	}, []string{"role", "status"}) // status: allowed, denied
)
