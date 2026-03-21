package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/damn8daniel/observability-platform/internal/alerting"
	"github.com/damn8daniel/observability-platform/internal/api"
	"github.com/damn8daniel/observability-platform/internal/config"
	"github.com/damn8daniel/observability-platform/internal/grpcserver"
	"github.com/damn8daniel/observability-platform/internal/ingestion"
	"github.com/damn8daniel/observability-platform/internal/retention"
	"github.com/damn8daniel/observability-platform/internal/storage"
	"github.com/damn8daniel/observability-platform/internal/tenant"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Warn("failed to load config, using defaults", "error", err)
		cfg = config.DefaultConfig()
	}

	logger.Info("starting observability platform",
		"http_addr", cfg.Server.HTTPAddr,
		"grpc_addr", cfg.Server.GRPCAddr,
	)

	// Initialize ClickHouse
	store, err := storage.NewClickHouseStore(cfg.ClickHouse)
	if err != nil {
		logger.Error("failed to connect to ClickHouse", "error", err)
		os.Exit(1)
	}
	defer store.Close()

	// Run migrations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := store.Migrate(ctx); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	cancel()

	// Initialize tenant registry
	tenants := tenant.NewRegistry()
	_ = tenants.Register(&tenant.Tenant{
		ID:           "default",
		Name:         "Default Tenant",
		RateLimitRPS: 10000,
		MaxLogSize:   1 << 20, // 1MB
		Enabled:      true,
	})

	// Initialize ingestion buffers
	batchCfg := ingestion.DefaultBatchConfig()
	logBuf := ingestion.NewLogBuffer(store, batchCfg, logger)
	spanBuf := ingestion.NewSpanBuffer(store, batchCfg, logger)
	metricBuf := ingestion.NewMetricBuffer(store, batchCfg, logger)

	// Initialize alerting engine
	alertEngine := alerting.NewEngine(store, cfg.Alerting, logger)
	go alertEngine.Start()

	// Initialize retention cleaner
	cleaner := retention.NewCleaner(store, cfg.Retention, logger)
	go cleaner.Start()

	// Start gRPC server
	grpcSrv := grpcserver.New(
		cfg.GRPC,
		logBuf,
		spanBuf,
		metricBuf,
		tenants,
		cfg.Tenancy,
		logger,
	)
	go func() {
		if err := grpcSrv.Serve(cfg.Server.GRPCAddr); err != nil {
			logger.Error("gRPC server failed", "error", err)
		}
	}()

	// Start HTTP server
	router := api.NewRouter(*cfg, store, logBuf, spanBuf, metricBuf, alertEngine, logger)
	httpSrv := &http.Server{
		Addr:         cfg.Server.HTTPAddr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info("HTTP server listening", "addr", cfg.Server.HTTPAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server failed", "error", err)
		}
	}()

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Info("received shutdown signal", "signal", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop accepting new requests
	grpcSrv.GracefulStop()
	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error("HTTP shutdown error", "error", err)
	}

	// Flush buffers
	logBuf.Stop()
	spanBuf.Stop()
	metricBuf.Stop()

	// Stop background workers
	alertEngine.Stop()
	cleaner.Stop()

	logger.Info("shutdown complete")
	fmt.Println("goodbye")
}
