package grpcserver

import (
	"log/slog"
	"net"

	pb "github.com/damn8daniel/observability-platform/proto/gen"
	"github.com/damn8daniel/observability-platform/internal/config"
	"github.com/damn8daniel/observability-platform/internal/ingestion"
	"github.com/damn8daniel/observability-platform/internal/middleware"
	"github.com/damn8daniel/observability-platform/internal/tenant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// Server wraps the gRPC server and all ingestion services.
type Server struct {
	grpcServer *grpc.Server
	listener   net.Listener
	logger     *slog.Logger
}

// New creates a configured gRPC server with all services registered.
func New(
	cfg config.GRPCConfig,
	logBuf *ingestion.LogBuffer,
	spanBuf *ingestion.SpanBuffer,
	metricBuf *ingestion.MetricBuffer,
	tenants *tenant.Registry,
	tenantCfg config.TenancyConfig,
	logger *slog.Logger,
) *Server {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.MaxSendMsgSize),
		grpc.ChainUnaryInterceptor(
			middleware.GRPCLoggingInterceptor(logger),
			middleware.GRPCTenantInterceptor(tenantCfg),
			middleware.GRPCRecoveryInterceptor(logger),
		),
		grpc.ChainStreamInterceptor(
			middleware.GRPCStreamLoggingInterceptor(logger),
			middleware.GRPCStreamRecoveryInterceptor(logger),
		),
	}

	srv := grpc.NewServer(opts...)

	// Register services
	pb.RegisterLogIngestionServiceServer(srv, NewLogIngestionServer(logBuf, tenants, logger))
	pb.RegisterTraceIngestionServiceServer(srv, NewTraceIngestionServer(spanBuf, tenants, logger))
	pb.RegisterMetricsIngestionServiceServer(srv, NewMetricsIngestionServer(metricBuf, tenants, logger))

	// Enable reflection for grpcurl / debugging
	reflection.Register(srv)

	return &Server{
		grpcServer: srv,
		logger:     logger,
	}
}

// Serve starts listening on the configured address.
func (s *Server) Serve(addr string) error {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = lis
	s.logger.Info("gRPC server listening", "addr", addr)
	return s.grpcServer.Serve(lis)
}

// GracefulStop gracefully stops the gRPC server.
func (s *Server) GracefulStop() {
	s.grpcServer.GracefulStop()
}
