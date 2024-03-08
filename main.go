package main

import (
	"context"
	"os"

	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/robbert229/jaeger-postgresql/internal/store"

	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "jaeger-postgresql",
		Level: hclog.Warn, // Jaeger only captures >= Warn, so don't bother logging below Warn
	})

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("invalid database url")
		os.Exit(1)
	}

	err := sql.Migrate(logger, databaseURL)
	if err != nil {
		logger.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}

	pgxconfig, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		logger.Error("failed to parse database url", "error", err)
		os.Exit(1)
	}

	pgxconfig.MaxConns = 50

	pool, err := pgxpool.NewWithConfig(context.Background(), pgxconfig)
	if err != nil {
		logger.Error("failed to connect to the postgres database", "error", err)
		os.Exit(1)
	}

	store, err := store.NewStore(pool, logger)
	if err != nil {
		logger.Error("failed to open store", "error", err)
		os.Exit(1)
	}

	grpc.Serve(&shared.PluginServices{
		Store:        store,
		ArchiveStore: nil,
	})

	pool.Close()
}
