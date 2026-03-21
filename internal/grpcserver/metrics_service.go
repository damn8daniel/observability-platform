package grpcserver

import (
	"context"
	"io"
	"log/slog"
	"time"

	pb "github.com/damn8daniel/observability-platform/proto/gen"
	"github.com/damn8daniel/observability-platform/internal/ingestion"
	"github.com/damn8daniel/observability-platform/internal/storage"
	"github.com/damn8daniel/observability-platform/internal/tenant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MetricsIngestionServer implements the gRPC MetricsIngestionService.
type MetricsIngestionServer struct {
	pb.UnimplementedMetricsIngestionServiceServer

	metricBuffer *ingestion.MetricBuffer
	tenants      *tenant.Registry
	logger       *slog.Logger
}

// NewMetricsIngestionServer creates a new gRPC metrics ingestion server.
func NewMetricsIngestionServer(mb *ingestion.MetricBuffer, tr *tenant.Registry, logger *slog.Logger) *MetricsIngestionServer {
	return &MetricsIngestionServer{
		metricBuffer: mb,
		tenants:      tr,
		logger:       logger,
	}
}

// PushMetrics accepts a batch of metric records.
func (s *MetricsIngestionServer) PushMetrics(ctx context.Context, req *pb.PushMetricsRequest) (*pb.PushMetricsResponse, error) {
	if len(req.Metrics) == 0 {
		return &pb.PushMetricsResponse{}, nil
	}

	var accepted, rejected int64
	samples := make([]storage.MetricSample, 0, len(req.Metrics))

	for _, rec := range req.Metrics {
		sample := protoToMetricSample(rec)
		samples = append(samples, sample)
		accepted++
	}

	if len(samples) > 0 {
		s.metricBuffer.PushBatch(samples)
	}

	return &pb.PushMetricsResponse{
		Accepted: accepted,
		Rejected: rejected,
	}, nil
}

// StreamMetrics accepts a stream of individual metric records.
func (s *MetricsIngestionServer) StreamMetrics(stream pb.MetricsIngestionService_StreamMetricsServer) error {
	var accepted, rejected int64

	for {
		rec, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.PushMetricsResponse{
				Accepted: accepted,
				Rejected: rejected,
			})
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive: %v", err)
		}

		sample := protoToMetricSample(rec)
		s.metricBuffer.Push(sample)
		accepted++
	}
}

func protoToMetricSample(rec *pb.MetricRecord) storage.MetricSample {
	ts := time.Now()
	if rec.Timestamp != nil {
		ts = rec.Timestamp.AsTime()
	}

	return storage.MetricSample{
		TenantID:  rec.TenantId,
		Name:      rec.Name,
		Value:     rec.Value,
		Timestamp: ts,
		Labels:    rec.Labels,
		Type:      storage.MetricType(rec.Type),
	}
}
