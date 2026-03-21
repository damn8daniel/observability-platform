package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// LogsIngested tracks the total number of log entries ingested.
	LogsIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "observability",
		Subsystem: "logs",
		Name:      "ingested_total",
		Help:      "Total number of log entries ingested",
	}, []string{"tenant_id", "service", "level"})

	// SpansIngested tracks the total number of spans ingested.
	SpansIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "observability",
		Subsystem: "traces",
		Name:      "spans_ingested_total",
		Help:      "Total number of spans ingested",
	}, []string{"tenant_id", "service"})

	// MetricSamplesIngested tracks the total number of metric samples ingested.
	MetricSamplesIngested = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "observability",
		Subsystem: "metrics",
		Name:      "samples_ingested_total",
		Help:      "Total number of metric samples ingested",
	}, []string{"tenant_id", "name"})

	// QueryDuration tracks the duration of query operations.
	QueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "observability",
		Subsystem: "query",
		Name:      "duration_seconds",
		Help:      "Duration of query operations in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"type", "tenant_id"})

	// GRPCRequestDuration tracks gRPC call latency.
	GRPCRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "observability",
		Subsystem: "grpc",
		Name:      "request_duration_seconds",
		Help:      "Duration of gRPC requests in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method"})

	// HTTPRequestDuration tracks HTTP request latency.
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "observability",
		Subsystem: "http",
		Name:      "request_duration_seconds",
		Help:      "Duration of HTTP requests in seconds",
		Buckets:   prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	// BatchFlushDuration tracks how long batch flushes take.
	BatchFlushDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "observability",
		Subsystem: "buffer",
		Name:      "flush_duration_seconds",
		Help:      "Duration of batch flush operations",
		Buckets:   prometheus.DefBuckets,
	}, []string{"type"})

	// AlertsFired tracks the total number of alerts fired.
	AlertsFired = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "observability",
		Subsystem: "alerts",
		Name:      "fired_total",
		Help:      "Total number of alerts fired",
	}, []string{"tenant_id", "rule_id"})

	// ActiveConnections tracks active gRPC and HTTP connections.
	ActiveConnections = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "observability",
		Name:      "active_connections",
		Help:      "Number of active connections",
	}, []string{"type"})
)
