package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNoTenant guards against running tenant-scoped work with an empty tenant.
var ErrNoTenant = errors.New("database: tenant id is required")

// Store owns the connection pool and hands out tenant-scoped query sets.
type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, url string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = 20
	cfg.MinConns = 2
	cfg.MaxConnLifetime = time.Hour
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return &Store{pool: pool}, nil
}

func (s *Store) Close() { s.pool.Close() }

func (s *Store) Health(ctx context.Context) error { return s.pool.Ping(ctx) }

// Unscoped runs queries with no tenant set, so row-level security denies
// every tenant-scoped row. Only provisioning and pre-auth lookups belong here.
func (s *Store) Unscoped() *Queries { return New(s.pool) }

// InTenant runs fn inside a transaction scoped to one tenant, rolling back on error.
func (s *Store) InTenant(ctx context.Context, tenantID uuid.UUID, fn func(*Queries) error) error {
	if tenantID == uuid.Nil {
		return ErrNoTenant
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Transaction-local, so it cannot leak to the next borrower of this connection.
	if _, err := tx.Exec(ctx, "SELECT set_config('app.tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("scope transaction to tenant: %w", err)
	}
	if err := fn(New(tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// IsNotFound reports whether err is pgx's empty-result sentinel.
func IsNotFound(err error) bool { return errors.Is(err, pgx.ErrNoRows) }
