package main

import (
	"context"
	"encoding/json"
	"flag"
	"os"

	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/robbert229/jaeger-postgresql/internal/store"

	"github.com/hashicorp/go-hclog"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
)

// Configuration is the main configuration struct for the github.com/robbert229/jaeger-postgresql plugin.
type Configuration struct {
	DatabaseURL string `json:"database_url"`
}

func main() {
	logger := hclog.New(&hclog.LoggerOptions{
		Name:  "github.com/robbert229/jaeger-postgresql",
		Level: hclog.Warn, // Jaeger only captures >= Warn, so don't bother logging below Warn
	})

	var configPath string
	flag.StringVar(&configPath, "config", "", "A path to the plugin's configuration file")
	flag.Parse()

	var config Configuration
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		logger.Error("failed to read configuration file", "error", err)
		os.Exit(1)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		logger.Error("failed to parse configuration", "error", err)
		os.Exit(1)
	}

	err = sql.Migrate(logger, config.DatabaseURL)
	if err != nil {
		logger.Error("failed to migrate database", "error", err)
		os.Exit(1)
	}

	pgxconfig, err := pgxpool.ParseConfig(config.DatabaseURL)
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
