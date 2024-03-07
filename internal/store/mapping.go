package store

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/robbert229/jaeger-postgresql/internal/sql"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jaegertracing/jaeger/model"
	"go.opentelemetry.io/otel/trace"
)

// DecodeTraceID converts a slice of raw bytes into a trace id.
func DecodeTraceID(raw []byte) model.TraceID {
	low := binary.LittleEndian.Uint64(raw[0:8])
	high := binary.LittleEndian.Uint64(raw[8:16])
	return model.NewTraceID(low, high)
}

// EncodeTraceID converts a trace id to a slice of raw bytes.
func EncodeTraceID(traceID model.TraceID) []byte {
	raw := []byte{}
	raw = binary.LittleEndian.AppendUint64(raw, traceID.High)
	raw = binary.LittleEndian.AppendUint64(raw, traceID.Low)
	return raw
}

// EncodeSpanID encodes a span id into a slice of bytes.
func EncodeSpanID(spanID model.SpanID) []byte {
	return binary.LittleEndian.AppendUint64(nil, uint64(spanID))
}

// DecodeSpanID decodes a span id form a byte slice.
func DecodeSpanID(raw []byte) model.SpanID {
	return model.NewSpanID(binary.LittleEndian.Uint64(raw))
}

func EncodeInterval(duration time.Duration) pgtype.Interval {
	return pgtype.Interval{Microseconds: duration.Microseconds(), Valid: true}
}

func EncodeTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t, Valid: true}
}

func encodeTagsToSlice(input []model.KeyValue) [][]any {
	slice := make([][]any, len(input))

	for i, kv := range input {
		var value interface{}
		if kv.VType == model.ValueType_STRING {
			value = kv.VStr
		} else if kv.VType == model.ValueType_BOOL {
			value = kv.VBool
		} else if kv.VType == model.ValueType_INT64 {
			value = fmt.Sprintf("%d", kv.VInt64)
		} else if kv.VType == model.ValueType_FLOAT64 {
			value = kv.VFloat64
		} else if kv.VType == model.ValueType_BINARY {
			value = base64.RawStdEncoding.EncodeToString(kv.VBinary)
		}

		slice[i] = []any{
			kv.Key,
			kv.VType,
			value,
		}
	}

	return slice
}

func EncodeTags(input []model.KeyValue) ([]byte, error) {
	slice := encodeTagsToSlice(input)

	bytes, err := json.Marshal(slice)
	if err != nil {
		return nil, fmt.Errorf("failed to encode to json: %w", err)
	}

	return bytes, nil
}

func decodeTagsFromSlice(slice []any) ([]model.KeyValue, error) {
	var tags []model.KeyValue
	for _, subslice := range slice {
		cast := subslice.([]any)

		key := cast[0].(string)
		vType := model.ValueType(int(cast[1].(float64)))
		value := cast[2]

		kv := model.KeyValue{Key: key, VType: vType}
		switch vType {
		case model.StringType:
			kv.VStr = value.(string)
		case model.BoolType:
			kv.VBool = value.(bool)
		case model.Int64Type:
			parsed, err := strconv.ParseInt(value.(string), 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int: %w", err)
			}

			kv.VInt64 = parsed
		case model.Float64Type:
			kv.VFloat64 = value.(float64)
		case model.BinaryType:
			bytes, err := base64.RawStdEncoding.DecodeString(value.(string))
			if err != nil {
				return nil, fmt.Errorf("failed to parse: %w", err)
			}

			kv.VBinary = bytes
		}

		tags = append(tags, kv)
	}

	// tags must not be nil. This is because the serialization/deserialization
	// tests demand it to be empty array
	if tags == nil {
		tags = []model.KeyValue{}
	}

	return tags, nil
}

func DecodeTags(input []byte) ([]model.KeyValue, error) {
	slice := []any{}
	if err := json.Unmarshal(input, &slice); err != nil {
		return nil, fmt.Errorf("failed to decode tag json: %w", err)
	}

	tags, err := decodeTagsFromSlice(slice)
	if err != nil {
		return nil, err
	}

	return tags, nil

}

func EncodeSpanKind(modelKind trace.SpanKind) sql.Spankind {
	switch modelKind {
	case trace.SpanKindClient:
		return sql.SpankindClient
	case trace.SpanKindServer:
		return sql.SpankindServer
	case trace.SpanKindProducer:
		return sql.SpankindProducer
	case trace.SpanKindConsumer:
		return sql.SpankindConsumer
	case trace.SpanKindInternal:
		return sql.SpankindInternal
	case trace.SpanKindUnspecified:
		return sql.SpankindUnspecified
	default:
		return sql.SpankindUnspecified
	}
}

func EncodeLogs(logs []model.Log) ([]byte, error) {
	slice := make([][]any, len(logs))
	for i, log := range logs {

		slice[i] = []any{
			pgtype.Timestamp{Time: log.Timestamp, Valid: true},
			encodeTagsToSlice(log.Fields),
		}
	}

	bytes, err := json.Marshal(slice)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func TruncateTime(ts time.Time) time.Time {
	ts, err := time.Parse(time.RFC3339Nano, ts.Format(time.RFC3339Nano))
	if err != nil {
		panic(err)
	}

	return ts
}

func DecodeLogs(raw []byte) ([]model.Log, error) {
	slice := [][]any{}
	if err := json.Unmarshal(raw, &slice); err != nil {
		return nil, fmt.Errorf("failed to decode logs json: %w", err)
	}

	logs := make([]model.Log, len(slice))
	for i, subslice := range slice {
		cast := subslice[1].([]any)
		fields, err := decodeTagsFromSlice(cast)
		if err != nil {
			return nil, err
		}

		layout := time.RFC3339Nano
		t, err := time.Parse(layout, subslice[0].(string))
		if err != nil {
			return nil, err
		}

		logs[i] = model.Log{
			Timestamp: t,
			Fields:    fields,
		}
	}

	return logs, nil
}

func EncodeSpanRefs(spanrefs []model.SpanRef) ([]byte, error) {
	if len(spanrefs) == 0 {
		return []byte("[]"), nil
	}

	slice := make([][]any, len(spanrefs))
	for i, spanref := range spanrefs {
		slice[i] = []any{
			base64.StdEncoding.EncodeToString(EncodeTraceID(spanref.TraceID)),
			base64.StdEncoding.EncodeToString(EncodeSpanID(spanref.SpanID)),
			int32(spanref.RefType),
		}
	}
	bytes, err := json.Marshal(slice)
	if err != nil {
		return nil, err
	}

	return bytes, nil
}

func DecodeSpanRefs(data []byte) ([]model.SpanRef, error) {
	var slice [][]any
	err := json.Unmarshal(data, &slice)
	if err != nil {
		return nil, err
	}

	results := make([]model.SpanRef, len(slice))

	for i, subslice := range slice {
		traceID, err := base64.StdEncoding.DecodeString(subslice[0].(string))
		if err != nil {
			return nil, err
		}

		spanID, err := base64.StdEncoding.DecodeString(subslice[1].(string))
		if err != nil {
			return nil, err
		}

		results[i] = model.SpanRef{
			TraceID: DecodeTraceID(traceID),
			SpanID:  DecodeSpanID(spanID),
			RefType: model.SpanRefType(int32(subslice[2].(float64))),
		}
	}

	return results, nil
}
