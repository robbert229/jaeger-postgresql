package store

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/robbert229/jaeger-postgresql/internal/sqltest"

	"github.com/stretchr/testify/require"

	"github.com/jaegertracing/jaeger/model"
	jaeger_integration_tests "github.com/jaegertracing/jaeger/plugin/storage/integration"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

func TestJaegerStorageIntegration(t *testing.T) {
	conn, cleanup, closer := sqltest.Harness(t)
	defer closer.Close()

	require.Nil(t, cleanup())

	q := sql.New(conn)

	logger := slog.Default()
	reader := NewReader(q, logger.With("component", "reader"))
	writer := NewWriter(q, logger.With("component", "writer"))
	si := jaeger_integration_tests.StorageIntegration{
		SpanReader:                   reader,
		SpanWriter:                   writer,
		DependencyReader:             reader,
		GetDependenciesReturnsSource: false,
		CleanUp:                      cleanup,
		Refresh:                      func() error { return nil },
		SkipList:                     []string{},
	}

	// Runs all storage integration tests.
	si.IntegrationTestAll(t)
}

func TestSpans(t *testing.T) {
	conn, cleanup, closer := sqltest.Harness(t)
	defer closer.Close()

	require.Nil(t, cleanup())

	ctx := context.Background()

	q := sql.New(conn)

	logger := slog.Default()
	w := NewWriter(q, logger)
	r := NewReader(q, logger)

	ts := TruncateTime(time.Now())

	span := &model.Span{
		TraceID:       model.NewTraceID(0, 0),
		SpanID:        model.NewSpanID(0),
		OperationName: "operation",
		Process:       model.NewProcess("service", []model.KeyValue{model.Bool("foo", true)}),
		Logs:          []model.Log{{Timestamp: ts, Fields: []model.KeyValue{model.Bool("foo", false)}}},
		Tags:          []model.KeyValue{model.Bool("fizzbuzz", true)},
		References:    []model.SpanRef{},
	}

	err := w.WriteSpan(ctx, span)
	require.Nil(t, err)

	trace, err := r.FindTraces(ctx, &spanstore.TraceQueryParameters{
		ServiceName:   "service",
		OperationName: "operation",
		NumTraces:     100,
	})
	require.Nil(t, err)

	require.Len(t, trace, 1)
	require.Equal(t, span, trace[0].Spans[0])
}
