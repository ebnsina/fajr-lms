// Package db embeds SQL migrations so the binary carries its own schema.
package db

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrations returns the migration files rooted at the directory goose expects.
func Migrations() fs.FS {
	sub, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		panic("db: embedded migrations missing: " + err.Error())
	}
	return sub
}
