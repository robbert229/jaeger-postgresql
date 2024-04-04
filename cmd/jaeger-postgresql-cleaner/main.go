package main

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/robbert229/jaeger-postgresql/internal/logger"
	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

// ProvideLogger returns a function that provides a logger
func ProvideLogger() any {
	return func(cfg Config) (*slog.Logger, error) {
		return logger.New(&cfg.LogLevel)
	}
}

// ProvidePgxPool returns a function that provides a pgx pool
func ProvidePgxPool() any {
	return func(cfg Config, logger *slog.Logger, lc fx.Lifecycle) (*pgxpool.Pool, error) {
		databaseURL := cfg.Database.URL
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
			if cfg.Database.MaxConns == 0 {
				maxConns = 20
			} else {
				maxConns = int32(cfg.Database.MaxConns)
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
func clean(ctx context.Context, pool *pgxpool.Pool, maxAge time.Duration) (int64, error) {
	q := sql.New(pool)
	result, err := q.CleanSpans(ctx, pgtype.Timestamp{Time: time.Now().Add(-1 * maxAge), Valid: true})
	if err != nil {
		return 0, err
	}

	return result, nil
}

type Config struct {
	Database struct {
		URL      string `mapstructure:"url"`
		MaxConns int    `mapstructure:"max-conns"`
	} `mapstructure:"database"`

	LogLevel string `mapstructure:"log-level"`

	MaxSpanAge time.Duration `mapstructure:"max-span-age"`
}

func ProvideConfig() func() (Config, error) {
	return func() (Config, error) {
		pflag.String("database.url", "", "the postgres connection url to use to connect to the database")
		pflag.Int("database.max-conns", 20, "Max number of database connections of which the plugin will try to maintain at any given time")
		pflag.String("log-level", "warn", "Minimal allowed log level")
		pflag.Duration("max-span-age", time.Hour*24, "Maximum age of a span before it will be cleaned")

		v := viper.New()
		v.SetEnvPrefix("JAEGER_POSTGRESQL")
		v.AutomaticEnv()
		v.SetConfigFile("jaeger-postgresql")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))
		pflag.Parse()
		v.BindPFlags(pflag.CommandLine)

		var cfg Config
		if err := v.ReadInConfig(); err != nil {
			_, ok := err.(*fs.PathError)
			_, ok2 := err.(viper.ConfigFileNotFoundError)

			if !ok && !ok2 {
				return cfg, fmt.Errorf("failed to read in config: %w", err)
			}
		}

		err := v.Unmarshal(&cfg)
		if err != nil {
			return cfg, fmt.Errorf("failed to decode configuration: %w", err)
		}

		return cfg, nil
	}
}

func main() {
	fx.New(
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger.With("component", "uber/fx")}
		}),
		fx.Provide(
			ProvideConfig(),
			ProvideLogger(),
			ProvidePgxPool(),
		),
		fx.Invoke(func(cfg Config, pool *pgxpool.Pool, lc fx.Lifecycle, logger *slog.Logger, stopper fx.Shutdowner) error {
			go func(ctx context.Context) {
				ctx, cancelFn := context.WithTimeout(ctx, time.Minute)
				defer cancelFn()

				count, err := clean(ctx, pool, cfg.MaxSpanAge)
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
