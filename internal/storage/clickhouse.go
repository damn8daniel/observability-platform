package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/damn8daniel/observability-platform/internal/config"
)

// ClickHouseStore implements log, trace, and metric storage backed by ClickHouse.
type ClickHouseStore struct {
	db *sql.DB
}

// NewClickHouseStore establishes a ClickHouse connection pool.
func NewClickHouseStore(cfg config.ClickHouseConfig) (*ClickHouseStore, error) {
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: cfg.Addrs,
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		MaxOpenConns: 20,
		MaxIdleConns: 10,
		ConnMaxLifetime: 10 * time.Minute,
	})

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("clickhouse ping: %w", err)
	}

	return &ClickHouseStore{db: db}, nil
}

// Migrate creates the required tables and indices.
func (s *ClickHouseStore) Migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS logs (
			id          String,
			tenant_id   String,
			timestamp   DateTime64(3),
			level       LowCardinality(String),
			service     LowCardinality(String),
			message     String,
			trace_id    String DEFAULT '',
			span_id     String DEFAULT '',
			attributes  Map(String, String),
			INDEX idx_message message TYPE tokenbf_v1(10240, 3, 0) GRANULARITY 4
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (tenant_id, service, timestamp)
		TTL toDateTime(timestamp) + INTERVAL 30 DAY`,

		`CREATE TABLE IF NOT EXISTS spans (
			trace_id       String,
			span_id        String,
			parent_span_id String DEFAULT '',
			tenant_id      String,
			service        LowCardinality(String),
			operation      String,
			start_time     DateTime64(6),
			end_time       DateTime64(6),
			duration_ns    Int64,
			status         UInt8,
			attributes     Map(String, String),
			events         Nested(
				name       String,
				timestamp  DateTime64(6),
				attrs      Map(String, String)
			)
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(start_time)
		ORDER BY (tenant_id, service, trace_id, start_time)
		TTL toDateTime(start_time) + INTERVAL 7 DAY`,

		`CREATE TABLE IF NOT EXISTS metrics (
			tenant_id  String,
			name       LowCardinality(String),
			value      Float64,
			timestamp  DateTime64(3),
			labels     Map(String, String),
			type       UInt8
		) ENGINE = MergeTree()
		PARTITION BY toYYYYMM(timestamp)
		ORDER BY (tenant_id, name, timestamp)
		TTL toDateTime(timestamp) + INTERVAL 90 DAY`,

		`CREATE TABLE IF NOT EXISTS alert_rules (
			id         String,
			tenant_id  String,
			name       String,
			query      String,
			type       LowCardinality(String),
			condition  String,
			threshold  Float64,
			duration   Int64,
			channels   Array(String),
			enabled    UInt8,
			created_at DateTime64(3)
		) ENGINE = ReplacingMergeTree()
		ORDER BY (tenant_id, id)`,

		`CREATE TABLE IF NOT EXISTS alerts (
			id          String,
			rule_id     String,
			tenant_id   String,
			status      LowCardinality(String),
			message     String,
			fired_at    DateTime64(3),
			resolved_at Nullable(DateTime64(3))
		) ENGINE = MergeTree()
		ORDER BY (tenant_id, fired_at)`,

		`CREATE TABLE IF NOT EXISTS dashboards (
			id         String,
			tenant_id  String,
			name       String,
			panels     String,
			created_at DateTime64(3),
			updated_at DateTime64(3)
		) ENGINE = ReplacingMergeTree(updated_at)
		ORDER BY (tenant_id, id)`,
	}

	for _, q := range queries {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return fmt.Errorf("migration query failed: %w", err)
		}
	}

	return nil
}

// InsertLogs batch-inserts log entries into ClickHouse.
func (s *ClickHouseStore) InsertLogs(ctx context.Context, logs []LogEntry) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO logs (id, tenant_id, timestamp, level, service, message, trace_id, span_id, attributes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, l := range logs {
		if _, err := stmt.ExecContext(ctx,
			l.ID, l.TenantID, l.Timestamp, l.Level, l.Service,
			l.Message, l.TraceID, l.SpanID, l.Attributes,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec: %w", err)
		}
	}

	return tx.Commit()
}

// InsertSpans batch-inserts spans into ClickHouse.
func (s *ClickHouseStore) InsertSpans(ctx context.Context, spans []Span) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO spans (trace_id, span_id, parent_span_id, tenant_id, service, operation,
		 start_time, end_time, duration_ns, status, attributes) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, sp := range spans {
		if _, err := stmt.ExecContext(ctx,
			sp.TraceID, sp.SpanID, sp.ParentSpanID, sp.TenantID, sp.Service,
			sp.Operation, sp.StartTime, sp.EndTime, sp.Duration.Nanoseconds(),
			int(sp.Status), sp.Attributes,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec: %w", err)
		}
	}

	return tx.Commit()
}

// InsertMetrics batch-inserts metric samples.
func (s *ClickHouseStore) InsertMetrics(ctx context.Context, samples []MetricSample) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx,
		"INSERT INTO metrics (tenant_id, name, value, timestamp, labels, type) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("prepare: %w", err)
	}
	defer stmt.Close()

	for _, m := range samples {
		if _, err := stmt.ExecContext(ctx,
			m.TenantID, m.Name, m.Value, m.Timestamp, m.Labels, int(m.Type),
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("exec: %w", err)
		}
	}

	return tx.Commit()
}

// QueryLogs searches logs with full-text search, field filtering, and time ranges.
func (s *ClickHouseStore) QueryLogs(ctx context.Context, q LogQuery) ([]LogEntry, int64, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "tenant_id = ?")
	args = append(args, q.TenantID)

	if q.Query != "" {
		conditions = append(conditions, "message ILIKE ?")
		args = append(args, "%"+q.Query+"%")
	}
	if q.Level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, q.Level)
	}
	if q.Service != "" {
		conditions = append(conditions, "service = ?")
		args = append(args, q.Service)
	}
	if q.TraceID != "" {
		conditions = append(conditions, "trace_id = ?")
		args = append(args, q.TraceID)
	}
	if !q.StartTime.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, q.StartTime)
	}
	if !q.EndTime.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, q.EndTime)
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	countQuery := fmt.Sprintf("SELECT count() FROM logs WHERE %s", where)
	var total int64
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count query: %w", err)
	}

	// Fetch results
	orderBy := "timestamp"
	if q.OrderBy != "" {
		orderBy = q.OrderBy
	}
	orderDir := "DESC"
	if q.OrderDir != "" {
		orderDir = q.OrderDir
	}
	limit := 100
	if q.Limit > 0 {
		limit = q.Limit
	}

	selectQuery := fmt.Sprintf(
		`SELECT id, tenant_id, timestamp, level, service, message, trace_id, span_id, attributes
		 FROM logs WHERE %s ORDER BY %s %s LIMIT %d OFFSET %d`,
		where, orderBy, orderDir, limit, q.Offset,
	)

	rows, err := s.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("select query: %w", err)
	}
	defer rows.Close()

	var results []LogEntry
	for rows.Next() {
		var entry LogEntry
		if err := rows.Scan(
			&entry.ID, &entry.TenantID, &entry.Timestamp, &entry.Level,
			&entry.Service, &entry.Message, &entry.TraceID, &entry.SpanID,
			&entry.Attributes,
		); err != nil {
			return nil, 0, fmt.Errorf("scan: %w", err)
		}
		results = append(results, entry)
	}

	return results, total, rows.Err()
}

// QueryTraces returns spans matching the given trace query.
func (s *ClickHouseStore) QueryTraces(ctx context.Context, q TraceQuery) ([]Span, error) {
	var conditions []string
	var args []interface{}

	conditions = append(conditions, "tenant_id = ?")
	args = append(args, q.TenantID)

	if q.TraceID != "" {
		conditions = append(conditions, "trace_id = ?")
		args = append(args, q.TraceID)
	}
	if q.Service != "" {
		conditions = append(conditions, "service = ?")
		args = append(args, q.Service)
	}
	if q.Operation != "" {
		conditions = append(conditions, "operation ILIKE ?")
		args = append(args, "%"+q.Operation+"%")
	}
	if q.MinDuration > 0 {
		conditions = append(conditions, "duration_ns >= ?")
		args = append(args, q.MinDuration.Nanoseconds())
	}
	if q.MaxDuration > 0 {
		conditions = append(conditions, "duration_ns <= ?")
		args = append(args, q.MaxDuration.Nanoseconds())
	}
	if !q.StartTime.IsZero() {
		conditions = append(conditions, "start_time >= ?")
		args = append(args, q.StartTime)
	}
	if !q.EndTime.IsZero() {
		conditions = append(conditions, "start_time <= ?")
		args = append(args, q.EndTime)
	}
	if q.Status != nil {
		conditions = append(conditions, "status = ?")
		args = append(args, int(*q.Status))
	}

	where := strings.Join(conditions, " AND ")
	limit := 100
	if q.Limit > 0 {
		limit = q.Limit
	}

	query := fmt.Sprintf(
		`SELECT trace_id, span_id, parent_span_id, tenant_id, service, operation,
		 start_time, end_time, duration_ns, status, attributes
		 FROM spans WHERE %s ORDER BY start_time ASC LIMIT %d`,
		where, limit,
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query spans: %w", err)
	}
	defer rows.Close()

	var spans []Span
	for rows.Next() {
		var sp Span
		var durationNs int64
		var status int
		if err := rows.Scan(
			&sp.TraceID, &sp.SpanID, &sp.ParentSpanID, &sp.TenantID,
			&sp.Service, &sp.Operation, &sp.StartTime, &sp.EndTime,
			&durationNs, &status, &sp.Attributes,
		); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		sp.Duration = time.Duration(durationNs)
		sp.Status = SpanStatus(status)
		spans = append(spans, sp)
	}

	return spans, rows.Err()
}

// AggregateLogsByField groups logs and returns counts per unique value of the specified field.
func (s *ClickHouseStore) AggregateLogsByField(ctx context.Context, tenantID, field string, start, end time.Time, limit int) (*AggregationResult, error) {
	allowedFields := map[string]bool{"level": true, "service": true}
	if !allowedFields[field] {
		return nil, fmt.Errorf("aggregation on field %q not allowed", field)
	}

	query := fmt.Sprintf(
		`SELECT %s AS key, count() AS cnt FROM logs
		 WHERE tenant_id = ? AND timestamp >= ? AND timestamp <= ?
		 GROUP BY key ORDER BY cnt DESC LIMIT ?`,
		field,
	)

	rows, err := s.db.QueryContext(ctx, query, tenantID, start, end, limit)
	if err != nil {
		return nil, fmt.Errorf("aggregate query: %w", err)
	}
	defer rows.Close()

	result := &AggregationResult{}
	for rows.Next() {
		var b AggBucket
		if err := rows.Scan(&b.Key, &b.Count); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		result.Buckets = append(result.Buckets, b)
		result.Total += b.Count
	}

	return result, rows.Err()
}

// GetCorrelatedData finds logs and traces linked by a trace ID.
func (s *ClickHouseStore) GetCorrelatedData(ctx context.Context, tenantID, traceID string) ([]LogEntry, []Span, error) {
	logs, _, err := s.QueryLogs(ctx, LogQuery{
		TenantID: tenantID,
		TraceID:  traceID,
		Limit:    1000,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("correlated logs: %w", err)
	}

	spans, err := s.QueryTraces(ctx, TraceQuery{
		TenantID: tenantID,
		TraceID:  traceID,
		Limit:    1000,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("correlated spans: %w", err)
	}

	return logs, spans, nil
}

// DeleteOldData removes data older than the specified TTL.
func (s *ClickHouseStore) DeleteOldData(ctx context.Context, table string, tsCol string, before time.Time) error {
	query := fmt.Sprintf("ALTER TABLE %s DELETE WHERE %s < ?", table, tsCol)
	_, err := s.db.ExecContext(ctx, query, before)
	return err
}

// Close closes the underlying database connection.
func (s *ClickHouseStore) Close() error {
	return s.db.Close()
}
