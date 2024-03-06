package store

import (
	"testing"

	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/robbert229/jaeger-postgresql/internal/sqltest"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	jaeger_integration_tests "github.com/jaegertracing/jaeger/plugin/storage/integration"
)

func TestJaegerStorageIntegration(t *testing.T) {
	conn, cleanup := sqltest.Harness(t)
	require.Nil(t, cleanup())

	q := sql.New(conn)

	logger := hclog.Default()
	reader := NewReader(q, logger.With("component", "reader"))
	writer := NewWriter(q, logger.With("component", "writer"))
	si := jaeger_integration_tests.StorageIntegration{
		SpanReader:                   reader,
		SpanWriter:                   writer,
		DependencyReader:             reader,
		GetDependenciesReturnsSource: false,
		CleanUp:                      cleanup,
		Refresh:                      func() error { return nil },
		SkipList: []string{
			//"TestJaegerStorageIntegration/GetTrace",
			"TestJaegerStorageIntegration/GetLargeSpans",
			//"TestJaegerStorageIntegration/FindTraces",
		},
	}
	// Runs all storage integration tests.
	si.IntegrationTestAll(t)
}
