package store

import (
	"github.com/robbert229/jaeger-postgresql/internal/sql"

	hclog "github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
)

var (
	_ shared.StoragePlugin = (*Store)(nil)
)

type Store struct {
	reader *Reader
	writer *Writer
}

func NewStore(pool *pgxpool.Pool, logger hclog.Logger) (*Store, error) {
	q := sql.New(pool)

	store := &Store{
		reader: NewReader(q, logger),
		writer: NewWriter(q, logger),
	}

	return store, nil
}

func (s *Store) SpanReader() spanstore.Reader {
	return s.reader
}

func (s *Store) SpanWriter() spanstore.Writer {
	return s.writer
}

func (s *Store) DependencyReader() dependencystore.Reader {
	return s.reader
}
