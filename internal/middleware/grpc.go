package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/damn8daniel/observability-platform/internal/config"
	"github.com/damn8daniel/observability-platform/internal/tenant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// GRPCLoggingInterceptor logs each unary RPC call.
func GRPCLoggingInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		level := slog.LevelInfo
		if err != nil {
			level = slog.LevelError
		}

		logger.Log(ctx, level, "gRPC call",
			"method", info.FullMethod,
			"duration", duration,
			"error", err,
		)

		return resp, err
	}
}

// GRPCStreamLoggingInterceptor logs each stream RPC.
func GRPCStreamLoggingInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		start := time.Now()
		err := handler(srv, ss)
		duration := time.Since(start)

		logger.Info("gRPC stream",
			"method", info.FullMethod,
			"duration", duration,
			"error", err,
		)

		return err
	}
}

// GRPCTenantInterceptor extracts tenant ID from gRPC metadata.
func GRPCTenantInterceptor(cfg config.TenancyConfig) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		if !cfg.Enabled {
			ctx = tenant.WithTenantID(ctx, cfg.DefaultTenant)
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		tenantIDs := md.Get(cfg.HeaderName)
		if len(tenantIDs) == 0 {
			if cfg.DefaultTenant != "" {
				ctx = tenant.WithTenantID(ctx, cfg.DefaultTenant)
				return handler(ctx, req)
			}
			return nil, status.Error(codes.Unauthenticated, "missing tenant ID")
		}

		ctx = tenant.WithTenantID(ctx, tenantIDs[0])
		return handler(ctx, req)
	}
}

// GRPCRecoveryInterceptor recovers from panics in unary handlers.
func GRPCRecoveryInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC panic recovered",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Errorf(codes.Internal, "internal error: %v", r)
			}
		}()

		return handler(ctx, req)
	}
}

// GRPCStreamRecoveryInterceptor recovers from panics in stream handlers.
func GRPCStreamRecoveryInterceptor(logger *slog.Logger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("gRPC stream panic recovered",
					"method", info.FullMethod,
					"panic", r,
					"stack", string(debug.Stack()),
				)
				err = status.Errorf(codes.Internal, "internal error: %v", r)
			}
		}()

		return handler(srv, ss)
	}
}

// Ensure fmt is used
var _ = fmt.Sprintf
