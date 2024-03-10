package store

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robbert229/jaeger-postgresql/internal/sql"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var _ spanstore.Reader = (*Reader)(nil)

// Reader can query for and load traces from PostgreSQL v2.x.
type Reader struct {
	logger *slog.Logger
	q      *sql.Queries
}

// NewReader returns a new SpanReader for PostgreSQL v2.x.
func NewReader(q *sql.Queries, logger *slog.Logger) *Reader {
	return &Reader{
		q:      q,
		logger: logger,
	}
}

// GetServices returns all services traced by Jaeger
func (r *Reader) GetServices(ctx context.Context) ([]string, error) {
	services, err := r.q.GetServices(ctx)
	if err != nil {
		return nil, err
	}

	return services, nil
}

// GetOperations returns all operations for a specific service traced by Jaeger
func (r *Reader) GetOperations(ctx context.Context, param spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	response, err := r.q.GetOperations(ctx, param.ServiceName)
	if err != nil {
		return nil, err
	}

	var operations = make([]spanstore.Operation, len(response))
	for i, iter := range response {
		operations[i] = spanstore.Operation{Name: iter.Name, SpanKind: string(iter.Kind)}
	}

	return operations, nil
}

// GetTrace takes a traceID and returns a Trace associated with that traceID
func (r *Reader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	{
		promGetTraceCounter.Inc()

		start := time.Now()
		defer func() {
			promGetTraceHistogram.Observe(time.Since(start).Seconds())
		}()
	}

	dbSpans, err := r.q.GetTraceSpans(ctx, EncodeTraceID(traceID))
	if err != nil {
		return nil, fmt.Errorf("failed to get trace spans: %w", err)
	}

	if len(dbSpans) == 0 {
		return nil, fmt.Errorf("trace not found")
	}

	var spans []*model.Span = make([]*model.Span, len(dbSpans))
	for i, dbSpan := range dbSpans {
		tags, err := DecodeTags(dbSpan.Tags)
		if err != nil {
			return nil, fmt.Errorf("failed to decode span tags: %w", err)
		}

		processTags, err := DecodeTags(dbSpan.ProcessTags)
		if err != nil {
			return nil, fmt.Errorf("failed to decode process tags: %w", err)
		}

		duration := time.Duration(dbSpan.Duration.Microseconds * 1000)

		logs, err := DecodeLogs(dbSpan.Logs)
		if err != nil {
			return nil, fmt.Errorf("failed to decode logs: %w", err)
		}

		decodedSpanRefs, err := DecodeSpanRefs(dbSpan.Refs)
		if err != nil {
			return nil, fmt.Errorf("failed to decode spanrefs: %w", err)
		}

		spans[i] = &model.Span{
			TraceID:       DecodeTraceID(dbSpan.TraceID),
			SpanID:        DecodeSpanID(dbSpan.SpanID),
			OperationName: dbSpan.OperationName,
			Tags:          tags,
			References:    decodedSpanRefs,
			Flags:         model.Flags(int32(dbSpan.Flags)),
			StartTime:     dbSpan.StartTime.Time,
			Duration:      duration,
			Logs:          logs,
			Process: &model.Process{
				ServiceName: dbSpan.ProcessName,
				Tags:        processTags,
			},
			ProcessID: dbSpan.ProcessID,
			Warnings:  dbSpan.Warnings,
		}
	}

	return &model.Trace{
		Spans: spans,
	}, nil
}

// FindTraces retrieve traces that match the traceQuery
func (r *Reader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	{
		promFindTracesCounter.Inc()

		start := time.Now()
		defer func() {
			promFindTracesHistogram.Observe(time.Since(start).Seconds())
		}()
	}

	response, err := r.q.FindTraceIDs(ctx, sql.FindTraceIDsParams{
		ServiceName:            query.ServiceName,
		ServiceNameEnable:      len(query.ServiceName) > 0,
		OperationName:          query.OperationName,
		OperationNameEnable:    len(query.OperationName) > 0,
		StartTimeMinimum:       EncodeTimestamp(query.StartTimeMin),
		StartTimeMinimumEnable: query.StartTimeMin.After(time.Time{}),
		StartTimeMaximum:       EncodeTimestamp(query.StartTimeMax),
		StartTimeMaximumEnable: query.StartTimeMax.After(time.Time{}),
		DurationMinimum:        EncodeInterval(query.DurationMin),
		DurationMinimumEnable:  query.DurationMin != time.Duration(0),
		DurationMaximum:        EncodeInterval(query.DurationMax),
		DurationMaximumEnable:  query.DurationMax != time.Duration(0),
		NumTraces:              query.NumTraces,
		// Tags
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query trace ids: %w", err)
	}

	var traces []*model.Trace
	for _, id := range response {
		trace, err := r.GetTrace(ctx, DecodeTraceID(id))
		if err != nil {
			return nil, err
		}

		traces = append(traces, trace)
	}

	return traces, nil
}

// FindTraceIDs retrieve traceIDs that match the traceQuery
func (r *Reader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	{
		promFindTraceIDsCounter.Inc()

		start := time.Now()
		defer func() {
			promFindTraceIDsHistogram.Observe(time.Since(start).Seconds())
		}()
	}

	response, err := r.q.FindTraceIDs(ctx, sql.FindTraceIDsParams{
		ServiceName:            query.ServiceName,
		ServiceNameEnable:      len(query.ServiceName) > 0,
		OperationName:          query.OperationName,
		OperationNameEnable:    len(query.OperationName) > 0,
		StartTimeMinimum:       EncodeTimestamp(query.StartTimeMin),
		StartTimeMinimumEnable: query.StartTimeMin.After(time.Time{}),
		StartTimeMaximum:       EncodeTimestamp(query.StartTimeMax),
		StartTimeMaximumEnable: query.StartTimeMax.After(time.Time{}),
		DurationMinimum:        EncodeInterval(query.DurationMin),
		DurationMinimumEnable:  query.DurationMin > 0*time.Second,
		DurationMaximum:        EncodeInterval(query.DurationMax),
		DurationMaximumEnable:  query.DurationMax > 0*time.Second,
		// TODO(johnrowl) add tags
		NumTraces: query.NumTraces,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query trace ids: %w", err)
	}

	var traceIDs = make([]model.TraceID, len(response))
	for i, iter := range response {
		traceIDs[i] = DecodeTraceID(iter)
	}

	return traceIDs, nil
}

// GetDependencies returns all inter-service dependencies
func (r *Reader) GetDependencies(ctx context.Context, endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	return []model.DependencyLink{}, nil
}
