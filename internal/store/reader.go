package store

import (
	"context"
	"fmt"
	"time"

	"github.com/robbert229/jaeger-postgresql/internal/sql"

	hclog "github.com/hashicorp/go-hclog"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var _ spanstore.Reader = (*Reader)(nil)

// Reader can query for and load traces from PostgreSQL v2.x.
type Reader struct {
	logger hclog.Logger
	q      *sql.Queries
}

// NewReader returns a new SpanReader for PostgreSQL v2.x.
func NewReader(q *sql.Queries, logger hclog.Logger) *Reader {
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

	// return nil, fmt.Errorf("not implemented")
	// builder := &whereBuilder{where: "", params: make([]interface{}, 0)}

	// if traceID.Low > 0 {
	// 	builder.andWhere(traceID.Low, "trace_id_low = ?")
	// }
	// if traceID.High > 0 {
	// 	builder.andWhere(traceID.Low, "trace_id_high = ?")
	// }

	// var spans []Span
	// err := r.db.Model(&spans).Where(builder.where, builder.params...).Limit(1).Select()
	// ret := make([]*model.Span, 0, len(spans))
	// ret2 := make([]model.Trace_ProcessMapping, 0, len(spans))
	// for _, span := range spans {
	// 	ret = append(ret, toModelSpan(span))
	// 	ret2 = append(ret2, model.Trace_ProcessMapping{
	// 		ProcessID: span.ProcessID,
	// 		Process: model.Process{
	// 			ServiceName: span.Service.ServiceName,
	// 			Tags:        mapToModelKV(span.ProcessTags),
	// 		},
	// 	})
	// }

	// return &model.Trace{Spans: ret, ProcessMap: ret2}, err
}

// FindTraces retrieve traces that match the traceQuery
func (r *Reader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
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

	// traceIDs, err := r.FindTraceIDs(ctx, query)
	// ret := make([]*model.Trace, 0, len(traceIDs))
	// if err != nil {
	// 	return ret, err
	// }
	// grouping := make(map[model.TraceID]*model.Trace)
	// //idsLow := make([]uint64, 0, len(traceIDs))
	// for _, traceID := range traceIDs {
	// 	//idsLow = append(idsLow, traceID.Low)
	// 	var spans []Span
	// 	err = r.db.Model(&spans).Where("trace_id_low = ?", traceID.Low /*TODO high*/).
	// 		//Join("JOIN operations AS operation ON operation.id = span.operation_id").
	// 		//Join("JOIN services AS service ON service.id = span.service_id").
	// 		Relation("Operation").Relation("Service").Order("start_time ASC").Select()
	// 	if err != nil {
	// 		return ret, err
	// 	}
	// 	for _, span := range spans {
	// 		modelSpan := toModelSpan(span)
	// 		trace, found := grouping[modelSpan.TraceID]
	// 		if !found {
	// 			trace = &model.Trace{
	// 				Spans:      make([]*model.Span, 0, len(spans)),
	// 				ProcessMap: make([]model.Trace_ProcessMapping, 0, len(spans)),
	// 			}
	// 			grouping[modelSpan.TraceID] = trace
	// 		}
	// 		trace.Spans = append(trace.Spans, modelSpan)
	// 		procMap := model.Trace_ProcessMapping{
	// 			ProcessID: span.ProcessID,
	// 			Process: model.Process{
	// 				ServiceName: span.Service.ServiceName,
	// 				Tags:        mapToModelKV(span.ProcessTags),
	// 			},
	// 		}
	// 		trace.ProcessMap = append(trace.ProcessMap, procMap)
	// 	}
	// }

	// for _, trace := range grouping {
	// 	ret = append(ret, trace)
	// }

	// return ret, err
}

// FindTraceIDs retrieve traceIDs that match the traceQuery
func (r *Reader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	response, err := r.q.FindTraceIDs(ctx, sql.FindTraceIDsParams{
		ServiceName:            query.ServiceName,
		ServiceNameEnable:      len(query.ServiceName) > 0,
		OperationName:          query.OperationName,
		OperationNameEnable:    len(query.OperationName) > 0,
		StartTimeMinimum:       EncodeTimestamp(query.StartTimeMin),
		StartTimeMinimumEnable: query.StartTimeMin.After(time.Time{}),
		StartTimeMaximum:       EncodeTimestamp(query.StartTimeMax),
		// StartTimeMaximumEnable: query.StartTimeMax.After(time.Time{}),
		StartTimeMaximumEnable: false, // maintaining feature parity for now.
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
	// builder := buildTraceWhere(query)

	// limit := query.NumTraces
	// if limit <= 0 {
	// 	limit = 10
	// }

	// err = r.db.Model((*Span)(nil)).
	// 	Join("JOIN operations AS operation ON operation.id = span.operation_id").
	// 	Join("JOIN services AS service ON service.id = span.service_id").
	// 	ColumnExpr("distinct trace_id_low as Low, trace_id_high as High").
	// 	Where(builder.where, builder.params...).Limit(100 * limit).Select(&ret)

	// return ret, err
}

// GetDependencies returns all inter-service dependencies
func (r *Reader) GetDependencies(ctx context.Context, endTs time.Time, lookback time.Duration) ([]model.DependencyLink, error) {
	// response, err := r.q.GetDependencies(ctx)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to query dependencies: %w", err)
	// }

	// dependencies := make([]model.DependencyLink, len(response))
	// for i, iter := range response {
	// 	dependencies[i] = model.DependencyLink{
	// 		Parent:    iter.Parent,
	// 		Child:     iter.Child,
	// 		CallCount: uint64(iter.CallCount),
	// 		Source:    iter.Source,
	// 	}
	// }

	// return dependencies, nil

	return []model.DependencyLink{}, nil
}
