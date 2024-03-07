
-- +goose Up

CREATE TYPE SPANKIND AS ENUM ('server', 'client', 'unspecified', 'producer', 'consumer', 'ephemeral', 'internal');

CREATE TABLE services (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE operations (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    service_id BIGINT REFERENCES services(id) NOT NULL,
    kind SPANKIND NOT NULL,

    UNIQUE (name, kind, service_id)
);

-- here our primary key is not named id as is customary. Instead we have named
-- it hack_id. This is to help us differentiate it from span_id. We can't use
-- span_id as our primary key because its not guaranteed to be unique as a
-- compatability measure with zipkin.
CREATE TABLE spans (
  hack_id BIGSERIAL PRIMARY KEY,
  span_id BYTEA,
  trace_id BYTEA,
  operation_id BIGINT REFERENCES operations(id) NOT NULL,
  flags BIGINT,
  start_time TIMESTAMP WITH TIME ZONE NOT NULL,
  duration INTERVAL NOT NULL,
  tags JSONB NOT NULL,
  service_id BIGINT REFERENCES services(id) NOT NULL,
  process_id TEXT NOT NULL,
  process_tags JSONB NOT NULL,
  warnings TEXT[],
  logs JSONB,
  kind SPANKIND NOT NULL
);

CREATE TABLE spanrefs (
    id BIGSERIAL,
    PRIMARY KEY(id),
    source_span_id BYTEA NOT NULL,
    child_span_id BYTEA NOT NULL,
    trace_id BYTEA,
    ref_type TEXT NOT NULL
);

-- +goose Down

DROP TABLE spanrefs;
DROP TABLE spans;
DROP TABLE operations;
DROP TABLE services;

DROP TYPE SPANKIND;