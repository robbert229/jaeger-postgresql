package store

import (
	"context"
	"log/slog"
	"time"

	"github.com/jaegertracing/jaeger/model"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	promNamespace = "jaeger_postgresql"
)

// reader

var (

	// GetTrace
	promGetTraceCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "get_trace_total",
		Help:      "The total number of calls to GetTrace",
	})

	promGetTraceHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: promNamespace,
		Name:      "get_trace_seconds",
		Help:      "The time spent in GetTrace",
	})

	promGetTraceErrorsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "get_trace_errors_total",
		Help:      "The total number of errors returned from GetTrace",
	})

	// FindTraces
	promFindTracesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "find_traces_total",
		Help:      "The total number of calls to FindTraces",
	})

	promFindTracesHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: promNamespace,
		Name:      "find_traces_seconds",
		Help:      "The time spent in FindTraces",
	})

	promFindTracesErrorsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "find_traces_errors_total",
		Help:      "The total number of errors returned from FindTraces",
	})

	// FindTraceIDs
	promFindTraceIDsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "find_trace_ids_total",
		Help:      "The total number of calls to FindTraceIDs",
	})

	promFindTraceIDsHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: promNamespace,
		Name:      "find_trace_ids_seconds",
		Help:      "The time spent in FindTraceIDs",
	})

	promFindTraceIDsErrorsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "find_trace_ids_errors_total",
		Help:      "The total number of errors for FindTraceIDs",
	})
)

// writer

var (
	// WriteSpan
	promWriteSpanCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "write_span_total",
		Help:      "The total number of calls to WriteSpan",
	})

	promWriteSpanHistogram = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: promNamespace,
		Name:      "write_span_seconds",
		Help:      "The time spent in WriteSpans",
	})

	promWriteSpanErrorsCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: promNamespace,
		Name:      "write_span_errors_total",
		Help:      "The total number of errors returned from WriteSpan",
	})
)

// NewInstrumentedWriter returns a new spanstore.Writer that is instrumented.
func NewInstrumentedWriter(embedded spanstore.Writer, logger *slog.Logger) *InstrumentedWriter {
	return &InstrumentedWriter{Writer: embedded, logger: logger}
}

// InstrumentedWriter is a writer that has been instrumented.
type InstrumentedWriter struct {
	spanstore.Writer
	logger *slog.Logger
}

func (w InstrumentedWriter) WriteSpan(ctx context.Context, span *model.Span) error {
	// instrumentation preamble
	{
		promWriteSpanCounter.Inc()

		start := time.Now()
		defer func() {
			promWriteSpanHistogram.Observe(time.Since(start).Seconds())
		}()

		w.logger.Debug(
			"inserting span",
			"span", span.SpanID,
			"trace_id", span.TraceID,
			"operation_name", span.OperationName,
		)
	}

	err := w.Writer.WriteSpan(ctx, span)
	if err != nil {
		promWriteSpanErrorsCounter.Inc()
		w.logger.Error("failed to write span", "err", err)
		return err
	}

	return nil
}

// NewInstrumentedReader returns a new spanstore.Reader that is instrumented.
func NewInstrumentedReader(embedded spanstore.Reader, logger *slog.Logger) *InstrumentedReader {
	return &InstrumentedReader{Reader: embedded, logger: logger}
}

// InstrumentedReader is a reader that has been instrumented.
type InstrumentedReader struct {
	spanstore.Reader
	logger *slog.Logger
}

// GetTrace takes a traceID and returns a Trace associated with that traceID
func (r *InstrumentedReader) GetTrace(ctx context.Context, traceID model.TraceID) (*model.Trace, error) {
	{
		promGetTraceCounter.Inc()

		start := time.Now()
		defer func() {
			promGetTraceHistogram.Observe(time.Since(start).Seconds())
		}()
	}

	trace, err := r.Reader.GetTrace(ctx, traceID)
	if err != nil {
		promGetTraceErrorsCounter.Inc()
		r.logger.Error("failed to get trace", "err", err)
		return nil, err
	}

	return trace, nil
}

// FindTraces retrieve traces that match the traceQuery
func (r *InstrumentedReader) FindTraces(ctx context.Context, query *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	{
		promFindTracesCounter.Inc()

		start := time.Now()
		defer func() {
			promFindTracesHistogram.Observe(time.Since(start).Seconds())
		}()
	}

	traces, err := r.Reader.FindTraces(ctx, query)
	if err != nil {
		promFindTracesErrorsCounter.Inc()
		r.logger.Error("failed to find traces", "err", err)
		return nil, err
	}

	return traces, nil
}

// FindTraceIDs retrieve traceIDs that match the traceQuery
func (r *InstrumentedReader) FindTraceIDs(ctx context.Context, query *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	{
		promFindTraceIDsCounter.Inc()

		start := time.Now()
		defer func() {
			promFindTraceIDsErrorsCounter.Inc()
			promFindTraceIDsHistogram.Observe(time.Since(start).Seconds())
		}()
	}

	traceIDs, err := r.Reader.FindTraceIDs(ctx, query)
	if err != nil {
		r.logger.Error("failed to retrieve trace ids", "err", err)
		return nil, err
	}

	return traceIDs, nil
}
