package ingestion

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/damn8daniel/observability-platform/internal/storage"
)

// BatchConfig controls the batching behavior of the ingestion pipeline.
type BatchConfig struct {
	MaxBatchSize int
	FlushInterval time.Duration
}

// DefaultBatchConfig returns sensible batch defaults.
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MaxBatchSize:  1000,
		FlushInterval: 5 * time.Second,
	}
}

// LogBuffer accumulates log entries and flushes them in batches to ClickHouse.
type LogBuffer struct {
	store  *storage.ClickHouseStore
	cfg    BatchConfig
	logger *slog.Logger

	mu      sync.Mutex
	buffer  []storage.LogEntry
	done    chan struct{}
}

// NewLogBuffer creates a new buffered log writer.
func NewLogBuffer(store *storage.ClickHouseStore, cfg BatchConfig, logger *slog.Logger) *LogBuffer {
	lb := &LogBuffer{
		store:  store,
		cfg:    cfg,
		logger: logger,
		buffer: make([]storage.LogEntry, 0, cfg.MaxBatchSize),
		done:   make(chan struct{}),
	}
	go lb.flushLoop()
	return lb
}

// Push adds a log entry to the buffer. Non-blocking; will flush when batch is full.
func (lb *LogBuffer) Push(entry storage.LogEntry) {
	lb.mu.Lock()
	lb.buffer = append(lb.buffer, entry)
	shouldFlush := len(lb.buffer) >= lb.cfg.MaxBatchSize
	lb.mu.Unlock()

	if shouldFlush {
		lb.Flush()
	}
}

// PushBatch adds multiple log entries.
func (lb *LogBuffer) PushBatch(entries []storage.LogEntry) {
	lb.mu.Lock()
	lb.buffer = append(lb.buffer, entries...)
	shouldFlush := len(lb.buffer) >= lb.cfg.MaxBatchSize
	lb.mu.Unlock()

	if shouldFlush {
		lb.Flush()
	}
}

// Flush writes the current buffer to ClickHouse.
func (lb *LogBuffer) Flush() {
	lb.mu.Lock()
	if len(lb.buffer) == 0 {
		lb.mu.Unlock()
		return
	}
	batch := lb.buffer
	lb.buffer = make([]storage.LogEntry, 0, lb.cfg.MaxBatchSize)
	lb.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := lb.store.InsertLogs(ctx, batch); err != nil {
		lb.logger.Error("failed to flush log batch", "error", err, "batch_size", len(batch))
		// Re-queue failed batch
		lb.mu.Lock()
		lb.buffer = append(batch, lb.buffer...)
		lb.mu.Unlock()
	} else {
		lb.logger.Debug("flushed log batch", "batch_size", len(batch))
	}
}

func (lb *LogBuffer) flushLoop() {
	ticker := time.NewTicker(lb.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lb.Flush()
		case <-lb.done:
			lb.Flush() // Final flush
			return
		}
	}
}

// Stop gracefully shuts down the buffer, flushing remaining data.
func (lb *LogBuffer) Stop() {
	close(lb.done)
}

// SpanBuffer accumulates spans and flushes them in batches.
type SpanBuffer struct {
	store  *storage.ClickHouseStore
	cfg    BatchConfig
	logger *slog.Logger

	mu     sync.Mutex
	buffer []storage.Span
	done   chan struct{}
}

// NewSpanBuffer creates a new buffered span writer.
func NewSpanBuffer(store *storage.ClickHouseStore, cfg BatchConfig, logger *slog.Logger) *SpanBuffer {
	sb := &SpanBuffer{
		store:  store,
		cfg:    cfg,
		logger: logger,
		buffer: make([]storage.Span, 0, cfg.MaxBatchSize),
		done:   make(chan struct{}),
	}
	go sb.flushLoop()
	return sb
}

// Push adds a span to the buffer.
func (sb *SpanBuffer) Push(span storage.Span) {
	sb.mu.Lock()
	sb.buffer = append(sb.buffer, span)
	shouldFlush := len(sb.buffer) >= sb.cfg.MaxBatchSize
	sb.mu.Unlock()

	if shouldFlush {
		sb.Flush()
	}
}

// PushBatch adds multiple spans.
func (sb *SpanBuffer) PushBatch(spans []storage.Span) {
	sb.mu.Lock()
	sb.buffer = append(sb.buffer, spans...)
	shouldFlush := len(sb.buffer) >= sb.cfg.MaxBatchSize
	sb.mu.Unlock()

	if shouldFlush {
		sb.Flush()
	}
}

// Flush writes the current buffer to ClickHouse.
func (sb *SpanBuffer) Flush() {
	sb.mu.Lock()
	if len(sb.buffer) == 0 {
		sb.mu.Unlock()
		return
	}
	batch := sb.buffer
	sb.buffer = make([]storage.Span, 0, sb.cfg.MaxBatchSize)
	sb.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := sb.store.InsertSpans(ctx, batch); err != nil {
		sb.logger.Error("failed to flush span batch", "error", err, "batch_size", len(batch))
		sb.mu.Lock()
		sb.buffer = append(batch, sb.buffer...)
		sb.mu.Unlock()
	} else {
		sb.logger.Debug("flushed span batch", "batch_size", len(batch))
	}
}

func (sb *SpanBuffer) flushLoop() {
	ticker := time.NewTicker(sb.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sb.Flush()
		case <-sb.done:
			sb.Flush()
			return
		}
	}
}

// Stop gracefully shuts down the span buffer.
func (sb *SpanBuffer) Stop() {
	close(sb.done)
}

// MetricBuffer accumulates metric samples and flushes them in batches.
type MetricBuffer struct {
	store  *storage.ClickHouseStore
	cfg    BatchConfig
	logger *slog.Logger

	mu     sync.Mutex
	buffer []storage.MetricSample
	done   chan struct{}
}

// NewMetricBuffer creates a new buffered metric writer.
func NewMetricBuffer(store *storage.ClickHouseStore, cfg BatchConfig, logger *slog.Logger) *MetricBuffer {
	mb := &MetricBuffer{
		store:  store,
		cfg:    cfg,
		logger: logger,
		buffer: make([]storage.MetricSample, 0, cfg.MaxBatchSize),
		done:   make(chan struct{}),
	}
	go mb.flushLoop()
	return mb
}

// Push adds a metric sample.
func (mb *MetricBuffer) Push(sample storage.MetricSample) {
	mb.mu.Lock()
	mb.buffer = append(mb.buffer, sample)
	shouldFlush := len(mb.buffer) >= mb.cfg.MaxBatchSize
	mb.mu.Unlock()

	if shouldFlush {
		mb.Flush()
	}
}

// PushBatch adds multiple metric samples.
func (mb *MetricBuffer) PushBatch(samples []storage.MetricSample) {
	mb.mu.Lock()
	mb.buffer = append(mb.buffer, samples...)
	shouldFlush := len(mb.buffer) >= mb.cfg.MaxBatchSize
	mb.mu.Unlock()

	if shouldFlush {
		mb.Flush()
	}
}

// Flush writes the current buffer to ClickHouse.
func (mb *MetricBuffer) Flush() {
	mb.mu.Lock()
	if len(mb.buffer) == 0 {
		mb.mu.Unlock()
		return
	}
	batch := mb.buffer
	mb.buffer = make([]storage.MetricSample, 0, mb.cfg.MaxBatchSize)
	mb.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := mb.store.InsertMetrics(ctx, batch); err != nil {
		mb.logger.Error("failed to flush metric batch", "error", err, "batch_size", len(batch))
		mb.mu.Lock()
		mb.buffer = append(batch, mb.buffer...)
		mb.mu.Unlock()
	} else {
		mb.logger.Debug("flushed metric batch", "batch_size", len(batch))
	}
}

func (mb *MetricBuffer) flushLoop() {
	ticker := time.NewTicker(mb.cfg.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mb.Flush()
		case <-mb.done:
			mb.Flush()
			return
		}
	}
}

// Stop gracefully shuts down the metric buffer.
func (mb *MetricBuffer) Stop() {
	close(mb.done)
}
