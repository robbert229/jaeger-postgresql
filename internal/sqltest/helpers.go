package sqltest

import (
	"context"
	"fmt"

	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5"
	"github.com/robbert229/jaeger-postgresql/internal/sql"
)

func TruncateAll(conn *pgx.Conn) error {
	ctx := context.Background()
	tables := []string{"operations", "services", "spans"}
	for _, table := range tables {
		if _, err := conn.Exec(ctx, fmt.Sprintf("TRUNCATE %s CASCADE", table)); err != nil {
			return err
		}
	}

	return nil
}

func cleanup(conn *pgx.Conn) func() error {
	return func() error {
		err := TruncateAll(conn)
		if err != nil {
			return err
		}

		return nil
	}
}

// Harness provides a test harness
func Harness(t interface {
	Fatal(args ...any)
	Helper()
}) (*pgx.Conn, func() error) {
	t.Helper()

	err := sql.Migrate(hclog.NewNullLogger(), "postgres://postgres:password@localhost:5432/jaeger")
	if err != nil {
		t.Fatal("failed to migrate database", err)
	}

	conn, err := pgx.Connect(context.Background(), "postgres://postgres:password@localhost:5432/jaeger")
	if err != nil {
		t.Fatal("failed to connect to database", err)
	}

	return conn, cleanup(conn)
}
