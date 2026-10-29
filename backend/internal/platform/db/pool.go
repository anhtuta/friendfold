// Package db owns pgx connection pools (write + read).
package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/anhtuta/friendfold/backend/internal/platform/config"
)

// Pools holds the write (primary) and read (replica, or the same URL locally) pools.
type Pools struct {
	Write *pgxpool.Pool
	Read  *pgxpool.Pool
}

// New constructs write and read pools from config. It does not ping; a down
// database must not prevent the process from serving /v1/health.
func New(ctx context.Context, cfg config.Config) (*Pools, error) {
	write, err := newPool(ctx, cfg.DatabaseURL, cfg.PGXMaxConns)
	if err != nil {
		return nil, fmt.Errorf("db write: %w", err)
	}
	read, err := newPool(ctx, cfg.DatabaseReadURL, cfg.PGXMaxConns)
	if err != nil {
		write.Close()
		return nil, fmt.Errorf("db read: %w", err)
	}
	return &Pools{Write: write, Read: read}, nil
}

// Close releases both pools.
func (p *Pools) Close() {
	if p == nil {
		return
	}
	if p.Write != nil {
		p.Write.Close()
	}
	if p.Read != nil && p.Read != p.Write {
		p.Read.Close()
	}
}

func newPool(ctx context.Context, url string, maxConns int32) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	return pool, nil
}
