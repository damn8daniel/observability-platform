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

// TraceIngestionServer implements the gRPC TraceIngestionService.
type TraceIngestionServer struct {
	pb.UnimplementedTraceIngestionServiceServer

	spanBuffer *ingestion.SpanBuffer
	tenants    *tenant.Registry
	logger     *slog.Logger
}

// NewTraceIngestionServer creates a new gRPC trace ingestion server.
func NewTraceIngestionServer(sb *ingestion.SpanBuffer, tr *tenant.Registry, logger *slog.Logger) *TraceIngestionServer {
	return &TraceIngestionServer{
		spanBuffer: sb,
		tenants:    tr,
		logger:     logger,
	}
}

// IngestSpans accepts a batch of span records.
func (s *TraceIngestionServer) IngestSpans(ctx context.Context, req *pb.IngestSpansRequest) (*pb.IngestSpansResponse, error) {
	if len(req.Spans) == 0 {
		return &pb.IngestSpansResponse{}, nil
	}

	var accepted, rejected int64
	var errors []string
	spans := make([]storage.Span, 0, len(req.Spans))

	for _, rec := range req.Spans {
		span, err := protoToSpan(rec)
		if err != nil {
			rejected++
			errors = append(errors, err.Error())
			continue
		}
		spans = append(spans, span)
		accepted++
	}

	if len(spans) > 0 {
		s.spanBuffer.PushBatch(spans)
	}

	return &pb.IngestSpansResponse{
		Accepted: accepted,
		Rejected: rejected,
		Errors:   errors,
	}, nil
}

// StreamSpans accepts a stream of individual span records.
func (s *TraceIngestionServer) StreamSpans(stream pb.TraceIngestionService_StreamSpansServer) error {
	var accepted, rejected int64

	for {
		rec, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.IngestSpansResponse{
				Accepted: accepted,
				Rejected: rejected,
			})
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive: %v", err)
		}

		span, err := protoToSpan(rec)
		if err != nil {
			rejected++
			continue
		}
		s.spanBuffer.Push(span)
		accepted++
	}
}

func protoToSpan(rec *pb.SpanRecord) (storage.Span, error) {
	startTime := time.Now()
	if rec.StartTime != nil {
		startTime = rec.StartTime.AsTime()
	}
	endTime := startTime
	if rec.EndTime != nil {
		endTime = rec.EndTime.AsTime()
	}

	span := storage.Span{
		TraceID:      rec.TraceId,
		SpanID:       rec.SpanId,
		ParentSpanID: rec.ParentSpanId,
		TenantID:     rec.TenantId,
		Service:      rec.Service,
		Operation:    rec.Operation,
		StartTime:    startTime,
		EndTime:      endTime,
		Duration:     time.Duration(rec.DurationNs),
		Status:       storage.SpanStatus(rec.Status),
		Attributes:   rec.Attributes,
	}

	for _, ev := range rec.Events {
		evTime := time.Now()
		if ev.Timestamp != nil {
			evTime = ev.Timestamp.AsTime()
		}
		span.Events = append(span.Events, storage.SpanEvent{
			Name:       ev.Name,
			Timestamp:  evTime,
			Attributes: ev.Attributes,
		})
	}

	return span, nil
}
