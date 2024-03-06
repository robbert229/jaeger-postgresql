package store

import (
	"testing"

	"github.com/jaegertracing/jaeger/model"

	"github.com/stretchr/testify/require"
)

func TestToDomainTraceID(t *testing.T) {
	traceID := model.NewTraceID(127318, 12489421)

	encoded := EncodeTraceID(traceID)
	decoded := DecodeTraceID(encoded)

	require.Equal(t, decoded, traceID)
}
