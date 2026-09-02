// Command migrate applies or rolls back database migrations.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/ebnsina/fajr-lms/db"
	"github.com/ebnsina/fajr-lms/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "migrate:", err)
		os.Exit(1)
	}
}

func run() error {
	cmd := "up"
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	// Migrations run as the owner, not as the RLS-restricted API role.
	url := os.Getenv("ADMIN_DATABASE_URL")
	if url == "" {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		url = cfg.DatabaseURL
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	sqlDB, err := sql.Open("pgx", url)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, db.Migrations())
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}
	defer provider.Close()

	switch cmd {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			return fmt.Errorf("apply migrations: %w", err)
		}
		report(results)
	case "down":
		result, err := provider.Down(ctx)
		if err != nil {
			return fmt.Errorf("roll back migration: %w", err)
		}
		report([]*goose.MigrationResult{result})
	case "status":
		status, err := provider.Status(ctx)
		if err != nil {
			return fmt.Errorf("read migration status: %w", err)
		}
		for _, s := range status {
			fmt.Printf("%-8s %s\n", s.State, s.Source.Path)
		}
	default:
		return errors.New("usage: migrate [up|down|status]")
	}
	return nil
}

func report(results []*goose.MigrationResult) {
	if len(results) == 0 {
		fmt.Println("no migrations to apply")
		return
	}
	for _, r := range results {
		fmt.Printf("%-6s %s (%s)\n", r.Direction, r.Source.Path, r.Duration)
	}
}
