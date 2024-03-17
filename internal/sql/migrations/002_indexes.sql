
-- +goose Up

CREATE INDEX IF NOT EXISTS idx_trace_id ON spans (trace_id);
CREATE INDEX IF NOT EXISTS idx_services_name ON services(name);
CREATE INDEX IF NOT EXISTS idx_operations_name ON operations(name);
CREATE INDEX IF NOT EXISTS idx_spans_operation_service ON spans(operation_id, service_id);
CREATE INDEX IF NOT EXISTS idx_spans_start_duration ON spans(start_time, duration);
CREATE INDEX IF NOT EXISTS idx_spans_start_time ON spans(start_time);
CREATE INDEX IF NOT EXISTS idx_spans_duration ON spans(duration);

-- +goose Down

DROP INDEX idx_trace_id ON spans (trace_id);
DROP INDEX idx_services_name ON services(name);
DROP INDEX idx_operations_name ON operations(name);
DROP INDEX idx_spans_operation_service ON spans(operation_id, service_id);
DROP INDEX idx_spans_start_duration ON spans(start_time, duration);
DROP INDEX idx_spans_start_time ON spans(start_time);
DROP INDEX idx_spans_duration ON spans(duration);
