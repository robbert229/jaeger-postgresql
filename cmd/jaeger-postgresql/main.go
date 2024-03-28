package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaegertracing/jaeger/plugin/storage/grpc/shared"
	"github.com/jaegertracing/jaeger/storage/dependencystore"
	"github.com/jaegertracing/jaeger/storage/spanstore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/robbert229/jaeger-postgresql/internal/logger"
	"github.com/robbert229/jaeger-postgresql/internal/sql"
	"github.com/robbert229/jaeger-postgresql/internal/store"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"google.golang.org/grpc"
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

// ProvideSpanStoreReader returns a function that provides a spanstore reader.
func ProvideSpanStoreReader() any {
	return func(pool *pgxpool.Pool, logger *slog.Logger) spanstore.Reader {
		q := sql.New(pool)
		return store.NewInstrumentedReader(store.NewReader(q, logger), logger)
	}
}

// ProvideSpanStoreWriter returns a function that provides a spanstore writer
func ProvideSpanStoreWriter() any {
	return func(pool *pgxpool.Pool, logger *slog.Logger) spanstore.Writer {
		q := sql.New(pool)
		return store.NewInstrumentedWriter(store.NewWriter(q, logger), logger)
	}
}

// ProvideDependencyStoreReader provides a dependencystore reader
func ProvideDependencyStoreReader() any {
	return func(pool *pgxpool.Pool, logger *slog.Logger) dependencystore.Reader {
		q := sql.New(pool)
		return store.NewReader(q, logger)
	}
}

// ProvideHandler provides a grpc handler.
func ProvideHandler() any {
	return func(reader spanstore.Reader, writer spanstore.Writer, dependencyReader dependencystore.Reader) *shared.GRPCHandler {
		handler := shared.NewGRPCHandler(&shared.GRPCHandlerStorageImpl{
			SpanReader:          func() spanstore.Reader { return reader },
			SpanWriter:          func() spanstore.Writer { return writer },
			DependencyReader:    func() dependencystore.Reader { return dependencyReader },
			ArchiveSpanReader:   func() spanstore.Reader { return nil },
			ArchiveSpanWriter:   func() spanstore.Writer { return nil },
			StreamingSpanWriter: func() spanstore.Writer { return nil },
		})

		return handler
	}
}

// ProvideGRPCServer provides a grpc server.
func ProvideGRPCServer() any {
	return func(lc fx.Lifecycle, logger *slog.Logger) (*grpc.Server, error) {
		srv := grpc.NewServer()

		if grpcHostPortFlag == nil {
			return nil, fmt.Errorf("invalid grpc-server.host-port")
		}

		if *grpcHostPortFlag == "" {
			return nil, fmt.Errorf("invalid grpc-server.host-port given: %s", *grpcHostPortFlag)
		}

		lis, err := net.Listen("tcp", *grpcHostPortFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to listen: %w", err)
		}

		logger.Info("grpc server started", "addr", *grpcHostPortFlag)

		lc.Append(fx.StartStopHook(
			func(ctx context.Context) error {
				go srv.Serve(lis)
				return nil
			},

			func(ctx context.Context) error {
				srv.GracefulStop()
				return lis.Close()
			},
		))

		return srv, nil
	}
}

// ProvideAdminServer provides the admin http server.
func ProvideAdminServer() any {
	return func(lc fx.Lifecycle, logger *slog.Logger) (*http.ServeMux, error) {
		mux := http.NewServeMux()

		srv := http.Server{
			Handler: mux,
		}

		if adminHttpHostPortFlag == nil {
			return nil, fmt.Errorf("no admin.http.host-port given")
		}

		if *adminHttpHostPortFlag == "" {
			return nil, fmt.Errorf("invalid admin.http.host-port given: %s", *adminHttpHostPortFlag)
		}

		lis, err := net.Listen("tcp", *adminHttpHostPortFlag)
		if err != nil {
			return nil, fmt.Errorf("failed to listen: %w", err)
		}

		logger.Info("admin server started", "addr", lis.Addr())

		lc.Append(fx.StartStopHook(
			func(ctx context.Context) error {
				go srv.Serve(lis)
				return nil
			},

			func(ctx context.Context) error {
				return srv.Shutdown(ctx)
			},
		))

		return mux, nil
	}
}

var (
	databaseURLFlag       = flag.String("database.url", "", "the postgres connection url to use to connect to the database")
	databaseMaxConnsFlag  = flag.Int("database.max-conns", 20, "Max number of database connections of which the plugin will try to maintain at any given time")
	loglevelFlag          = flag.String("log-level", "warn", "Minimal allowed log level")
	grpcHostPortFlag      = flag.String("grpc-server.host-port", ":12345", "the host:port (eg 127.0.0.1:12345 or :12345) of the storage provider's gRPC server")
	adminHttpHostPortFlag = flag.String("admin.http.host-port", ":12346", "The host:port (e.g. 127.0.0.1:12346 or :12346) for the admin server, including health check, /metrics, etc.")
)

var (
	spansTableDiskSizeGuage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jaeger_postgresql",
		Name:      "spans_table_bytes",
		Help:      "The size of the spans table in bytes",
	})

	spansGuage = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "jaeger_postgresql",
		Name:      "spans_count",
		Help:      "The number of spans",
	})
)

func main() {
	flag.Parse()

	fx.New(
		fx.WithLogger(func(logger *slog.Logger) fxevent.Logger {
			return &fxevent.SlogLogger{Logger: logger.With("component", "uber/fx")}
		}),
		fx.Provide(
			ProvideLogger(),
			ProvidePgxPool(),
			ProvideSpanStoreReader(),
			ProvideSpanStoreWriter(),
			ProvideDependencyStoreReader(),
			ProvideHandler(),
			ProvideGRPCServer(),
			ProvideAdminServer(),
		),
		fx.Invoke(func(srv *grpc.Server, handler *shared.GRPCHandler) error {
			return handler.Register(srv)
		}),
		fx.Invoke(func(conn *pgxpool.Pool, logger *slog.Logger, lc fx.Lifecycle) {
			ctx, cancelFn := context.WithCancel(context.Background())
			lc.Append(fx.StopHook(cancelFn))

			go func() {
				q := sql.New(conn)
				ticker := time.NewTicker(time.Second * 5)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						byteCount, err := q.GetSpansDiskSize(ctx)
						if err != nil {
							logger.Error("failed to query for disk size", "err", err)
							continue
						} else {
							spansTableDiskSizeGuage.Set(float64(byteCount))
						}

						count, err := q.GetSpansCount(ctx)
						if err != nil {
							logger.Error("failed to query for span count", "err", err)
							continue
						} else {
							spansGuage.Set(float64(count))
						}

					}
				}
			}()
		}),
		fx.Invoke(func(mux *http.ServeMux, conn *pgxpool.Pool) {
			mux.Handle("/metrics", promhttp.Handler())
			mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx, cancelFn := context.WithTimeout(r.Context(), time.Second*5)
				defer cancelFn()

				err := conn.Ping(ctx)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
				}

				w.WriteHeader(http.StatusOK)
			}))
		}),
	).Run()
}
