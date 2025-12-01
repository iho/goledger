package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics
type Metrics struct {
	// Transfer metrics
	TransfersCreated  prometheus.Counter
	TransfersReversed prometheus.Counter
	TransferDuration  prometheus.Histogram
	TransferAmount    prometheus.Histogram
	TransferErrors    *prometheus.CounterVec

	// Account metrics
	AccountsCreated   prometheus.Counter
	AccountBalance    *prometheus.GaugeVec
	AccountOperations *prometheus.CounterVec

	// Hold metrics
	HoldsCreated  prometheus.Counter
	HoldsVoided   prometheus.Counter
	HoldsCaptured prometheus.Counter
	HoldDuration  prometheus.Histogram

	// API metrics
	HTTPRequests *prometheus.CounterVec
	HTTPDuration *prometheus.HistogramVec
	GRPCRequests *prometheus.CounterVec
	GRPCDuration *prometheus.HistogramVec

	// Database metrics
	DBQueries     *prometheus.CounterVec
	DBDuration    *prometheus.HistogramVec
	DBConnections prometheus.Gauge
	DBErrors      *prometheus.CounterVec

	// Redis metrics
	RedisOperations *prometheus.CounterVec
	RedisDuration   *prometheus.HistogramVec
	RedisErrors     *prometheus.CounterVec

	// Authentication metrics
	AuthAttempts   *prometheus.CounterVec
	AuthFailures   *prometheus.CounterVec
	ActiveSessions prometheus.Gauge

	// Rate limiting metrics
	RateLimitHits *prometheus.CounterVec

	// Audit metrics
	AuditLogsCreated *prometheus.CounterVec
}

// New creates and registers all Prometheus metrics
func New() *Metrics {
	return &Metrics{
		// Transfer metrics
		TransfersCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "goledger_transfers_created_total",
			Help: "Total number of transfers created",
		}),
		TransfersReversed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "goledger_transfers_reversed_total",
			Help: "Total number of transfers reversed",
		}),
		TransferDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "goledger_transfer_duration_seconds",
			Help:    "Duration of transfer operations",
			Buckets: prometheus.DefBuckets,
		}),
		TransferAmount: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "goledger_transfer_amount",
			Help:    "Transfer amounts",
			Buckets: []float64{1, 10, 100, 1000, 10000, 100000, 1000000},
		}),
		TransferErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_transfer_errors_total",
				Help: "Total number of transfer errors by type",
			},
			[]string{"error_type"},
		),

		// Account metrics
		AccountsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "goledger_accounts_created_total",
			Help: "Total number of accounts created",
		}),
		AccountBalance: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "goledger_account_balance",
				Help: "Current account balance",
			},
			[]string{"account_id", "currency"},
		),
		AccountOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_account_operations_total",
				Help: "Total account operations by type",
			},
			[]string{"operation"},
		),

		// Hold metrics
		HoldsCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "goledger_holds_created_total",
			Help: "Total number of holds created",
		}),
		HoldsVoided: promauto.NewCounter(prometheus.CounterOpts{
			Name: "goledger_holds_voided_total",
			Help: "Total number of holds voided",
		}),
		HoldsCaptured: promauto.NewCounter(prometheus.CounterOpts{
			Name: "goledger_holds_captured_total",
			Help: "Total number of holds captured",
		}),
		HoldDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "goledger_hold_duration_seconds",
			Help:    "Duration of hold operations",
			Buckets: prometheus.DefBuckets,
		}),

		// API metrics
		HTTPRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_http_requests_total",
				Help: "Total HTTP requests",
			},
			[]string{"method", "path", "status"},
		),
		HTTPDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "goledger_http_duration_seconds",
				Help:    "HTTP request duration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
		GRPCRequests: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_grpc_requests_total",
				Help: "Total gRPC requests",
			},
			[]string{"method", "status"},
		),
		GRPCDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "goledger_grpc_duration_seconds",
				Help:    "gRPC request duration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method"},
		),

		// Database metrics
		DBQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_db_queries_total",
				Help: "Total database queries",
			},
			[]string{"operation", "table"},
		),
		DBDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "goledger_db_query_duration_seconds",
				Help:    "Database query duration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "table"},
		),
		DBConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "goledger_db_connections",
			Help: "Current number of database connections",
		}),
		DBErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_db_errors_total",
				Help: "Total database errors",
			},
			[]string{"operation"},
		),

		// Redis metrics
		RedisOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_redis_operations_total",
				Help: "Total Redis operations",
			},
			[]string{"operation"},
		),
		RedisDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "goledger_redis_duration_seconds",
				Help:    "Redis operation duration",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation"},
		),
		RedisErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_redis_errors_total",
				Help: "Total Redis errors",
			},
			[]string{"operation"},
		),

		// Authentication metrics
		AuthAttempts: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_auth_attempts_total",
				Help: "Total authentication attempts",
			},
			[]string{"status"},
		),
		AuthFailures: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_auth_failures_total",
				Help: "Total authentication failures",
			},
			[]string{"reason"},
		),
		ActiveSessions: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "goledger_active_sessions",
			Help: "Current number of active sessions",
		}),

		// Rate limiting metrics
		RateLimitHits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_rate_limit_hits_total",
				Help: "Total rate limit hits",
			},
			[]string{"ip"},
		),

		// Audit metrics
		AuditLogsCreated: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "goledger_audit_logs_total",
				Help: "Total audit logs created",
			},
			[]string{"action", "status"},
		),
	}
}
