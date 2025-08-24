// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os/signal"

	"github.com/adoublef/evetech/internal/database/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sys/unix"
)

type migrate struct {
	db struct {
		url  string // --database-url
		down bool   // --down
	}
}

func (c *migrate) parse(stdin io.Reader, getenv func(string) string, args ...string) error {
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.StringVar(&c.db.url, "database-url", "", "database url")  // cannot be empty
	fs.BoolVar(&c.db.down, "down", false, "migration direction") //

	err := fs.Parse(args)
	if err != nil {
		return err
	} else if fs.NArg() > 0 {
		fs.Usage()
		return flag.ErrHelp
	}
	return nil
}

func (c *migrate) run(ctx context.Context, stderr, stdout io.Writer) error {
	ctx, cancel := signal.NotifyContext(ctx, unix.SIGINT, unix.SIGKILL, unix.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, c.db.url)
	if err != nil {
		return fmt.Errorf("failed conntecting to postgres: %w", err)
	}
	defer pool.Close()

	var (
		f   func(context.Context, string) error
		dir string
	)
	if c.db.down {
		f = postgres.Down
		dir = "down"
	} else {
		f = postgres.Up
		dir = "up"
	}
	if err := f(ctx, c.db.url); err != nil {
		return fmt.Errorf("failed %s migration: %w", dir, err)
	}
	return nil
}
