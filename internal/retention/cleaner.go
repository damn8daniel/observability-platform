package retention

import (
	"context"
	"log/slog"
	"time"

	"github.com/damn8daniel/observability-platform/internal/config"
	"github.com/damn8daniel/observability-platform/internal/storage"
)

// Cleaner periodically removes data that has exceeded its TTL.
type Cleaner struct {
	store  *storage.ClickHouseStore
	cfg    config.RetentionConfig
	logger *slog.Logger
	done   chan struct{}
}

// NewCleaner creates a new retention cleaner.
func NewCleaner(store *storage.ClickHouseStore, cfg config.RetentionConfig, logger *slog.Logger) *Cleaner {
	return &Cleaner{
		store:  store,
		cfg:    cfg,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start begins the periodic cleanup loop.
func (c *Cleaner) Start() {
	interval := c.cfg.CleanupInterval
	if interval == 0 {
		interval = 1 * time.Hour
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	c.logger.Info("retention cleaner started",
		"logs_ttl", c.cfg.LogsTTL,
		"traces_ttl", c.cfg.TracesTTL,
		"metrics_ttl", c.cfg.MetricsTTL,
		"interval", interval,
	)

	// Run immediately on start
	c.cleanup()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.done:
			return
		}
	}
}

// Stop halts the cleanup loop.
func (c *Cleaner) Stop() {
	close(c.done)
}

func (c *Cleaner) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	now := time.Now()

	tables := []struct {
		name   string
		tsCol  string
		ttl    time.Duration
	}{
		{"logs", "timestamp", c.cfg.LogsTTL},
		{"spans", "start_time", c.cfg.TracesTTL},
		{"metrics", "timestamp", c.cfg.MetricsTTL},
	}

	for _, t := range tables {
		if t.ttl == 0 {
			continue
		}
		cutoff := now.Add(-t.ttl)
		if err := c.store.DeleteOldData(ctx, t.name, t.tsCol, cutoff); err != nil {
			c.logger.Error("retention cleanup failed",
				"table", t.name,
				"cutoff", cutoff,
				"error", err,
			)
		} else {
			c.logger.Info("retention cleanup completed",
				"table", t.name,
				"cutoff", cutoff,
			)
		}
	}
}
