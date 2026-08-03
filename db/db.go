package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"

	"linknest/config"
)

//go:embed migrations/001_init_mysql.sql
var migrationSQL string

func Open(cfg config.Config) (*sql.DB, error) {
	dsn, err := toDSN(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	pool.SetMaxOpenConns(cfg.MaxOpenConns)
	pool.SetMaxIdleConns(cfg.MaxIdleConns)
	pool.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := pool.PingContext(ctx); err != nil {
		_ = pool.Close()
		return nil, err
	}
	return pool, nil
}

// toDSN converts a mysql://user:pass@host:port/dbname?tls=true style URL
// (the format TiDB Cloud and most MySQL hosts publish) into the DSN string
// the go-sql-driver/mysql package expects.
func toDSN(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	cfg := mysql.NewConfig()
	cfg.User = u.User.Username()
	cfg.Passwd, _ = u.User.Password()
	cfg.Net = "tcp"
	cfg.Addr = u.Host
	cfg.DBName = strings.TrimPrefix(u.Path, "/")
	cfg.ParseTime = true
	cfg.Loc = time.UTC
	cfg.MultiStatements = true
	if u.Query().Get("tls") == "true" {
		cfg.TLSConfig = "true"
	}
	return cfg.FormatDSN(), nil
}

func Migrate(ctx context.Context, pool *sql.DB) error {
	if _, err := pool.ExecContext(ctx, migrationSQL); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	return nil
}
