package storage

import (
	"time"
)

// LogEntry represents a structured log record.
type LogEntry struct {
	ID         string            `json:"id"`
	TenantID   string            `json:"tenant_id"`
	Timestamp  time.Time         `json:"timestamp"`
	Level      string            `json:"level"`
	Service    string            `json:"service"`
	Message    string            `json:"message"`
	TraceID    string            `json:"trace_id,omitempty"`
	SpanID     string            `json:"span_id,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Span represents a single span in a distributed trace.
type Span struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	ParentSpanID string            `json:"parent_span_id,omitempty"`
	TenantID     string            `json:"tenant_id"`
	Service      string            `json:"service"`
	Operation    string            `json:"operation"`
	StartTime    time.Time         `json:"start_time"`
	EndTime      time.Time         `json:"end_time"`
	Duration     time.Duration     `json:"duration"`
	Status       SpanStatus        `json:"status"`
	Attributes   map[string]string `json:"attributes,omitempty"`
	Events       []SpanEvent       `json:"events,omitempty"`
}

// SpanStatus represents the status of a span.
type SpanStatus int

const (
	SpanStatusUnset SpanStatus = iota
	SpanStatusOK
	SpanStatusError
)

// SpanEvent is a time-stamped annotation on a span.
type SpanEvent struct {
	Name       string            `json:"name"`
	Timestamp  time.Time         `json:"timestamp"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// MetricSample represents a single metric data point.
type MetricSample struct {
	TenantID  string            `json:"tenant_id"`
	Name      string            `json:"name"`
	Value     float64           `json:"value"`
	Timestamp time.Time         `json:"timestamp"`
	Labels    map[string]string `json:"labels,omitempty"`
	Type      MetricType        `json:"type"`
}

// MetricType distinguishes metric kinds.
type MetricType int

const (
	MetricTypeGauge MetricType = iota
	MetricTypeCounter
	MetricTypeHistogram
	MetricTypeSummary
)

// LogQuery defines search parameters for log queries.
type LogQuery struct {
	TenantID  string
	Query     string
	Level     string
	Service   string
	TraceID   string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Offset    int
	OrderBy   string
	OrderDir  string
}

// TraceQuery defines search parameters for trace queries.
type TraceQuery struct {
	TenantID  string
	TraceID   string
	Service   string
	Operation string
	MinDuration time.Duration
	MaxDuration time.Duration
	StartTime time.Time
	EndTime   time.Time
	Limit     int
	Status    *SpanStatus
}

// AggregationResult holds the result of a log aggregation query.
type AggregationResult struct {
	Buckets []AggBucket `json:"buckets"`
	Total   int64       `json:"total"`
}

// AggBucket is a single bucket in an aggregation result.
type AggBucket struct {
	Key   string  `json:"key"`
	Count int64   `json:"count"`
	Value float64 `json:"value,omitempty"`
}

// AlertRule defines conditions that trigger alerts.
type AlertRule struct {
	ID          string        `json:"id"`
	TenantID    string        `json:"tenant_id"`
	Name        string        `json:"name"`
	Query       string        `json:"query"`
	Type        string        `json:"type"` // log, metric
	Condition   string        `json:"condition"`
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Channels    []string      `json:"channels"`
	Enabled     bool          `json:"enabled"`
	CreatedAt   time.Time     `json:"created_at"`
}

// Alert represents a fired alert.
type Alert struct {
	ID        string    `json:"id"`
	RuleID    string    `json:"rule_id"`
	TenantID  string    `json:"tenant_id"`
	Status    string    `json:"status"` // firing, resolved
	Message   string    `json:"message"`
	FiredAt   time.Time `json:"fired_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

// Dashboard represents a custom dashboard.
type Dashboard struct {
	ID        string          `json:"id"`
	TenantID  string          `json:"tenant_id"`
	Name      string          `json:"name"`
	Panels    []DashboardPanel `json:"panels"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// DashboardPanel represents a panel in a dashboard.
type DashboardPanel struct {
	ID       string            `json:"id"`
	Title    string            `json:"title"`
	Type     string            `json:"type"` // logs, metrics, traces
	Query    string            `json:"query"`
	Position PanelPosition     `json:"position"`
	Options  map[string]string `json:"options,omitempty"`
}

// PanelPosition defines the layout position of a panel.
type PanelPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}
