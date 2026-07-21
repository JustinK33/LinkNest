package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"linknest/internal/config"
)

func Open(cfg config.Config) (*sql.DB, error) {
	pool, err := sql.Open("pgx", cfg.DatabaseURL)
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

func Migrate(ctx context.Context, pool *sql.DB, path string) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	if _, err := pool.ExecContext(ctx, string(body)); err != nil {
		return fmt.Errorf("apply migration: %w", err)
	}
	return nil
}
