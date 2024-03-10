package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robbert229/fxslog"
	"github.com/robbert229/jaeger-postgresql/internal/logger"
	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"go.uber.org/fx"
)

var (
	databaseURLFlag      = flag.String("database.url", "", "the postgres connection url to use to connect to the database")
	databaseMaxConnsFlag = flag.Int("database.max-conns", 20, "Max number of database connections of which the plugin will try to maintain at any given time")
	loglevelFlag         = flag.String("log-level", "warn", "Minimal allowed log level")
	maxSpanAgeFlag       = flag.Duration("max-span-age", time.Hour*24, "Maximum age of a span before it will be cleaned")
)

// ProvideLogger returns a function that provides a logger
func ProvideLogger() any {
	return func() (*slog.Logger, error) {
		return logger.New(loglevelFlag)
	}
}

// ProvidePgxPool returns a function that provides a pgx pool
func ProvidePgxPool() any {
	return func(logger *slog.Logger, lc fx.Lifecycle) (*pgxpool.Pool, error) {
		if databaseURLFlag == nil {
			return nil, fmt.Errorf("invalid database url")
		}

		databaseURL := *databaseURLFlag
		if databaseURL == "" {
			return nil, fmt.Errorf("invalid database url")
		}

		err := sql.Migrate(logger, databaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to migrate database: %w", err)
		}

		pgxconfig, err := pgxpool.ParseConfig(databaseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse database url")
		}

		// handle max conns
		{
			var maxConns int32
			if databaseMaxConnsFlag == nil {
				maxConns = 20
			} else {
				maxConns = int32(*databaseMaxConnsFlag)
			}

			pgxconfig.MaxConns = maxConns
		}

		// handle timeout duration
		connectTimeoutDuration := time.Second * 10
		pgxconfig.ConnConfig.ConnectTimeout = connectTimeoutDuration

		ctx, cancelFn := context.WithTimeout(context.Background(), connectTimeoutDuration)
		defer cancelFn()

		pool, err := pgxpool.NewWithConfig(ctx, pgxconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to the postgres database: %w", err)
		}

		logger.Info("connected to postgres")

		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				pool.Close()
				return nil
			},
		})

		return pool, nil
	}
}

// clean purges the old roles from the database
func clean(ctx context.Context, pool *pgxpool.Pool) (int64, error) {
	q := sql.New(pool)
	result, err := q.CleanSpans(ctx, pgtype.Timestamp{Time: time.Now().Add(-1 * *maxSpanAgeFlag), Valid: true})
	if err != nil {
		return 0, err
	}

	return result, nil
}

func main() {
	flag.Parse()

	fx.New(
		fxslog.WithLogger(func(logger *slog.Logger) *slog.Logger {
			return logger.With("component", "uber/fx")
		}),
		fx.Provide(
			ProvideLogger(),
			ProvidePgxPool(),
		),
		fx.Invoke(func(pool *pgxpool.Pool, lc fx.Lifecycle, logger *slog.Logger, stopper fx.Shutdowner) error {
			go func(ctx context.Context) {
				ctx, cancelFn := context.WithTimeout(ctx, time.Minute)
				defer cancelFn()

				count, err := clean(ctx, pool)
				if err != nil {
					logger.Error("failed to clean database", "err", err)
					stopper.Shutdown(fx.ExitCode(1))
					return
				}

				logger.Info("successfully cleaned database", "spans", count)
				stopper.Shutdown(fx.ExitCode(0))
			}(context.Background())
			return nil
		}),
	).Run()
}
