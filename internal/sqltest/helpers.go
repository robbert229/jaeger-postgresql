package sqltest

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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

func getDatabaseURL() string {
	if url := os.Getenv("TEST_DATABASE_URL"); url != "" {
		return url
	}

	return "postgres://postgres:password@localhost:5432/jaeger"
}

// Harness provides a test harness
func Harness(t interface {
	Fatal(args ...any)
	Helper()
}) (*pgx.Conn, func() error) {
	t.Helper()

	err := sql.Migrate(slog.Default(), getDatabaseURL())
	if err != nil {
		t.Fatal("failed to migrate database", err)
	}

	conn, err := pgx.Connect(context.Background(), getDatabaseURL())
	if err != nil {
		t.Fatal("failed to connect to database", err)
	}

	return conn, cleanup(conn)
}
