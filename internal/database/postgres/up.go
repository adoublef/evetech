package postgres

import (
	"context"
	"embed"

	"go.adoublef.dev/database/sql/postgres"
)

//go:embed all:*.sql
var embedFS embed.FS

func Up(ctx context.Context, dsn string) error {
	return (&postgres.FS{URL: dsn, FS: embedFS}).Up(ctx)
}

func Down(ctx context.Context, dsn string) error {
	return (&postgres.FS{URL: dsn, FS: embedFS}).Down(ctx)
}
