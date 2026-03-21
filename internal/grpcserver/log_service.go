package grpcserver

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/damn8daniel/observability-platform/proto/gen"
	"github.com/damn8daniel/observability-platform/internal/ingestion"
	"github.com/damn8daniel/observability-platform/internal/storage"
	"github.com/damn8daniel/observability-platform/internal/tenant"
)

// LogIngestionServer implements the gRPC LogIngestionService.
type LogIngestionServer struct {
	pb.UnimplementedLogIngestionServiceServer

	logBuffer *ingestion.LogBuffer
	tenants   *tenant.Registry
	logger    *slog.Logger
}

// NewLogIngestionServer creates a new gRPC log ingestion server.
func NewLogIngestionServer(lb *ingestion.LogBuffer, tr *tenant.Registry, logger *slog.Logger) *LogIngestionServer {
	return &LogIngestionServer{
		logBuffer: lb,
		tenants:   tr,
		logger:    logger,
	}
}

// IngestLogs accepts a batch of log records.
func (s *LogIngestionServer) IngestLogs(ctx context.Context, req *pb.IngestLogsRequest) (*pb.IngestLogsResponse, error) {
	if len(req.Logs) == 0 {
		return &pb.IngestLogsResponse{}, nil
	}

	var accepted, rejected int64
	var errors []string
	entries := make([]storage.LogEntry, 0, len(req.Logs))

	for _, rec := range req.Logs {
		entry, err := protoToLogEntry(rec)
		if err != nil {
			rejected++
			errors = append(errors, err.Error())
			continue
		}
		entries = append(entries, entry)
		accepted++
	}

	if len(entries) > 0 {
		s.logBuffer.PushBatch(entries)
	}

	return &pb.IngestLogsResponse{
		Accepted: accepted,
		Rejected: rejected,
		Errors:   errors,
	}, nil
}

// StreamLogs accepts a stream of individual log records.
func (s *LogIngestionServer) StreamLogs(stream pb.LogIngestionService_StreamLogsServer) error {
	var accepted, rejected int64

	for {
		rec, err := stream.Recv()
		if err == io.EOF {
			return stream.SendAndClose(&pb.IngestLogsResponse{
				Accepted: accepted,
				Rejected: rejected,
			})
		}
		if err != nil {
			return status.Errorf(codes.Internal, "receive: %v", err)
		}

		entry, err := protoToLogEntry(rec)
		if err != nil {
			rejected++
			continue
		}
		s.logBuffer.Push(entry)
		accepted++
	}
}

// TailLogs streams matching logs in real-time (placeholder implementation).
func (s *LogIngestionServer) TailLogs(req *pb.TailLogsRequest, stream pb.LogIngestionService_TailLogsServer) error {
	// In production, this would subscribe to a message bus or internal channel.
	// For now, send a heartbeat every 5 seconds to keep the stream alive.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			// Heartbeat — would send real logs in full implementation
		}
	}
}

func protoToLogEntry(rec *pb.LogRecord) (storage.LogEntry, error) {
	id := rec.Id
	if id == "" {
		id = uuid.New().String()
	}

	ts := time.Now()
	if rec.Timestamp != nil {
		ts = rec.Timestamp.AsTime()
	}

	return storage.LogEntry{
		ID:         id,
		TenantID:   rec.TenantId,
		Timestamp:  ts,
		Level:      rec.Level,
		Service:    rec.Service,
		Message:    rec.Message,
		TraceID:    rec.TraceId,
		SpanID:     rec.SpanId,
		Attributes: rec.Attributes,
	}, nil
}

func logEntryToProto(entry storage.LogEntry) *pb.LogRecord {
	return &pb.LogRecord{
		Id:         entry.ID,
		TenantId:   entry.TenantID,
		Timestamp:  timestamppb.New(entry.Timestamp),
		Level:      entry.Level,
		Service:    entry.Service,
		Message:    entry.Message,
		TraceId:    entry.TraceID,
		SpanId:     entry.SpanID,
		Attributes: entry.Attributes,
	}
}
