package sqltest

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
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

type harnessCloser struct {
	*postgres.PostgresContainer
	*pgx.Conn
}

func (c harnessCloser) Close() error {
	err := c.Conn.Close(context.Background())
	if err != nil {
		return err
	}

	err = c.Terminate(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// Harness provides a test harness
func Harness(t interface {
	Errorf(format string, args ...interface{})
	FailNow()
	Helper()
}) (*pgx.Conn, func() error, io.Closer) {
	t.Helper()

	dbName := "jaeger"
	dbUser := "postgres"
	dbPassword := "password"

	ctx := context.Background()
	pgC, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15.2-alpine"),
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	require.Nil(t, err, "failed to get testcontainer")

	endpoint, err := pgC.Endpoint(ctx, "")
	require.Nil(t, err, "failed to get endpoint")

	databaseURL := "postgres://postgres:password@" + endpoint + "/postgres"

	err = sql.Migrate(slog.Default(), databaseURL)
	require.Nil(t, err, "failed to migrate database")

	conn, err := pgx.Connect(ctx, databaseURL)
	require.Nil(t, err, "failed to connect to database")

	return conn, func() error {
		return TruncateAll(conn)
	}, harnessCloser{pgC, conn}
}
