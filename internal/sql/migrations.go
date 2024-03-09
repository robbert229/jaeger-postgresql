package sql

import (
	"embed"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

var (
	mu sync.Mutex

	//go:embed migrations/*.sql
	migrations embed.FS
)

var _ goose.Logger = (*gooseLogger)(nil)

type gooseLogger struct {
	*slog.Logger
}

func (l *gooseLogger) Fatal(v ...interface{}) {
	l.Logger.Error(fmt.Sprint(v...))
}

func (l *gooseLogger) Fatalf(msg string, v ...interface{}) {
	l.Logger.Error(fmt.Sprintf(msg, v...))
}

func (l *gooseLogger) Print(v ...interface{}) {
	l.Logger.Info(fmt.Sprint(v...))
}

func (l *gooseLogger) Println(v ...interface{}) {
	l.Logger.Info(fmt.Sprint(v...))
}

func (l *gooseLogger) Printf(msg string, v ...interface{}) {
	trimmed := strings.Trim(msg, "\n")
	l.Logger.Info(fmt.Sprintf(trimmed, v...))
}

func Migrate(logger *slog.Logger, connStr string) error {
	mu.Lock()
	defer mu.Unlock()

	goose.SetLogger(&gooseLogger{logger})

	goose.SetBaseFS(migrations)

	goose.SetDialect("pgx")

	db, err := goose.OpenDBWithDriver("pgx", connStr)
	if err != nil {
		return fmt.Errorf("connecting to db for migrations: %w", err)
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting postgres dialect for migrations: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("unable to migrate database: %w", err)
	}

	return nil
}
