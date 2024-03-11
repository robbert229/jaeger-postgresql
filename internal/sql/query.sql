-- name: GetOperations :many
SELECT operations.name, operations.kind
FROM operations
  INNER JOIN services ON (operations.service_id = services.id)
WHERE services.name = sqlc.arg(service_name)::VARCHAR
ORDER BY operations.name ASC;

-- name: GetServices :many
SELECT services.name
FROM services
ORDER BY services.name ASC;

-- -- name: GetDependencies :many
-- SELECT
--     COUNT(*) AS call_count,
--     source_services.name as parent,
--     child_services.name as child,
--     '' as source
-- FROM spanrefs
--     INNER JOIN spans AS source_spans ON (source_spans.span_id = spanrefs.source_span_id)
--     INNER JOIN spans AS child_spans ON (child_spans.span_id = spanrefs.child_span_id)
--     INNER JOIN services AS source_services ON (source_spans.service_id = source_services.id)
--     INNER JOIN services AS child_services ON (child_spans.service_id = child_services.id)
-- GROUP BY source_services.name, child_services.name;

-- -- name: FindTraceIDs :many
-- SELECT DISTINCT spans.trace_id
-- FROM spans
--     INNER JOIN operations ON (operations.id = spans.operation_id)
--     INNER JOIN services ON (services.id = spans.service_id)
-- WHERE
--     (services.name = sqlc.arg(service_name)::VARCHAR OR sqlc.arg(service_name_enable)::BOOLEAN = FALSE) AND
--     (operations.name = sqlc.arg(operation_name)::VARCHAR OR sqlc.arg(operation_name_enable)::BOOLEAN = FALSE) AND
--     (start_time >= sqlc.arg(start_time_minimum)::TIMESTAMPTZ OR sqlc.arg(start_time_minimum_enable)::BOOLEAN = FALSE) AND
--     (start_time < sqlc.arg(start_time_maximum)::TIMESTAMPTZ OR sqlc.arg(start_time_maximum_enable)::BOOLEAN = FALSE) AND
--     (duration > sqlc.arg(duration_minimum)::INTERVAL OR sqlc.arg(duration_minimum_enable)::BOOLEAN = FALSE) AND
--     (duration < sqlc.arg(duration_maximum)::INTERVAL OR sqlc.arg(duration_maximum_enable)::BOOLEAN = FALSE)
-- ;
--LIMIT sqlc.arg(limit)::INT;

-- name: UpsertService :exec
INSERT INTO services (name) 
VALUES (sqlc.arg(name)::VARCHAR) ON CONFLICT(name) DO NOTHING RETURNING id;

-- name: GetServiceID :one
SELECT id
FROM services
WHERE name = sqlc.arg(name)::TEXT;

-- name: UpsertOperation :exec
INSERT INTO operations (name, service_id, kind) 
VALUES (
  sqlc.arg(name)::TEXT, 
  sqlc.arg(service_id)::BIGINT, 
  sqlc.arg(kind)::SPANKIND
) ON CONFLICT(name, service_id, kind) DO NOTHING RETURNING id;

-- name: GetOperationID :one
SELECT id 
FROM operations 
WHERE 
  name = sqlc.arg(name)::TEXT AND 
  service_id = sqlc.arg(service_id)::BIGINT AND 
  kind = sqlc.arg(kind)::SPANKIND;

-- name: GetTraceSpans :many
SELECT
  spans.span_id as span_id,
  spans.trace_id as trace_id,
  operations.name as operation_name,
  spans.flags as flags,
  spans.start_time as start_time,
  spans.duration as duration,
  spans.tags as tags,
  spans.process_id as process_id,
  spans.warnings as warnings,
  spans.kind as kind,
  services.name as process_name,
  spans.process_tags as process_tags,
  spans.logs as logs,
  spans.refs as refs
FROM spans 
  INNER JOIN operations ON (spans.operation_id = operations.id)
  INNER JOIN services ON (spans.service_id = services.id)
WHERE trace_id = sqlc.arg(trace_id)::BYTEA;

-- name: InsertSpan :one
INSERT INTO spans (
  span_id,
  trace_id,
  operation_id,
  flags,
  start_time,
  duration,
  tags,
  service_id,
  process_id,
  process_tags,
  warnings,
  kind,
  logs,
  refs
)
VALUES(
  sqlc.arg(span_id)::BYTEA,
  sqlc.arg(trace_id)::BYTEA,
  sqlc.arg(operation_id)::BIGINT,
  sqlc.arg(flags)::BIGINT,
  sqlc.arg(start_time)::TIMESTAMP,
  sqlc.arg(duration)::INTERVAL,
  sqlc.arg(tags)::JSONB,
  sqlc.arg(service_id)::BIGINT,
  sqlc.arg(process_id)::TEXT,
  sqlc.arg(process_tags)::JSONB,
  sqlc.arg(warnings)::TEXT[],
  sqlc.arg(kind)::SPANKIND,
  sqlc.arg(logs)::JSONB,
  sqlc.arg(refs)::JSONB
)
RETURNING spans.hack_id;

-- name: CleanSpans :execrows

DELETE FROM spans
WHERE spans.start_time < sqlc.arg(prune_before)::TIMESTAMP;

-- name: GetSpansDiskSize :one

SELECT pg_total_relation_size('spans');
