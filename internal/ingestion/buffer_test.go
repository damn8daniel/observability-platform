package ingestion

import (
	"testing"
	"time"
)

func TestDefaultBatchConfig(t *testing.T) {
	cfg := DefaultBatchConfig()

	if cfg.MaxBatchSize != 1000 {
		t.Errorf("MaxBatchSize = %d, want 1000", cfg.MaxBatchSize)
	}

	if cfg.FlushInterval != 5*time.Second {
		t.Errorf("FlushInterval = %v, want 5s", cfg.FlushInterval)
	}
}
