package store

import (
	"context"
	"fmt"
	"io"

	"github.com/robbert229/jaeger-postgresql/internal/sql"

	hclog "github.com/hashicorp/go-hclog"
	"go.opentelemetry.io/otel/trace"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var _ spanstore.Writer = (*Writer)(nil)
var _ io.Closer = (*Writer)(nil)

// Writer handles all writes to PostgreSQL 2.x for the Jaeger data model
type Writer struct {
	q      *sql.Queries
	logger hclog.Logger
}

// NewWriter returns a Writer.
func NewWriter(q *sql.Queries, logger hclog.Logger) *Writer {
	w := &Writer{
		q:      q,
		logger: logger,
	}

	return w
}

// Close triggers a graceful shutdown
func (w *Writer) Close() error {
	return nil
}

// WriteSpan saves the span into PostgreSQL
func (w *Writer) WriteSpan(ctx context.Context, span *model.Span) error {
	w.logger.Info(
		"inserting span",
		"span", span.SpanID,
		"trace_id", span.TraceID,
		"operation_name", span.OperationName,
	)
	err := w.q.UpsertService(ctx, span.Process.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to upsert span service: %w", err)
	}

	serviceID, err := w.q.GetServiceID(ctx, span.Process.ServiceName)
	if err != nil {
		return fmt.Errorf("failed to get service id: %w", err)
	}

	modelKind, ok := span.GetSpanKind()
	if !ok {
		modelKind = trace.SpanKindUnspecified
	}

	err = w.q.UpsertOperation(ctx, sql.UpsertOperationParams{
		Name:      span.OperationName,
		ServiceID: serviceID,
		Kind:      EncodeSpanKind(modelKind),
	})
	if err != nil {
		return fmt.Errorf("failed to upsert span operation: %w", err)
	}

	operationID, err := w.q.GetOperationID(ctx, sql.GetOperationIDParams{
		Name:      span.OperationName,
		ServiceID: serviceID,
		Kind:      EncodeSpanKind(modelKind),
	})
	if err != nil {
		return fmt.Errorf("failed to get operation id: %w", err)
	}

	logs, err := EncodeLogs(span.Logs)
	if err != nil {
		return fmt.Errorf("failed to encode logs: %w", err)
	}

	tags, err := EncodeTags(span.Tags)
	if err != nil {
		return fmt.Errorf("failed to encode tags: %w", err)
	}

	processTags, err := EncodeTags(span.Process.Tags)
	if err != nil {
		return fmt.Errorf("failed to encode process tags: %w", err)
	}

	encodedSpanRefs, err := EncodeSpanRefs(span.References)
	if err != nil {
		return fmt.Errorf("failed to encode spanrefs: %w", err)
	}

	_, err = w.q.InsertSpan(ctx, sql.InsertSpanParams{
		SpanID:      EncodeSpanID(span.SpanID),
		TraceID:     EncodeTraceID(span.TraceID),
		OperationID: operationID,
		Flags:       int64(span.Flags),
		StartTime:   EncodeTimestamp(span.StartTime),
		Duration:    EncodeInterval(span.Duration),
		Tags:        tags,
		ServiceID:   serviceID,
		ProcessID:   span.ProcessID,
		Warnings:    span.Warnings,
		ProcessTags: processTags,
		Kind:        EncodeSpanKind(modelKind),
		Logs:        logs,
		Refs:        encodedSpanRefs,
	})
	if err != nil {
		return fmt.Errorf("failed to insert span: %w", err)
	}

	return nil
}
