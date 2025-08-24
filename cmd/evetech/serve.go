// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"io"
	"net"
	"os/signal"
	"time"

	"github.com/adoublef/evetech/internal/net/http"
	o0 "github.com/adoublef/evetech/internal/order"
	o1 "github.com/adoublef/evetech/internal/order/v1"
	o2 "github.com/adoublef/evetech/internal/order/v2"
	o3 "github.com/adoublef/evetech/internal/order/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"
)

type serve struct {
	http struct {
		addr string // --http-address
	}
	db struct {
		url string // --database-url
	}
}

func (c *serve) parse(stdin io.Reader, getenv func(string) string, args ...string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.StringVar(&c.http.addr, "http-address", ":8080", "http address")
	fs.StringVar(&c.db.url, "database-url", "", "database url") // cannot be empty
	err := fs.Parse(args)
	if err != nil {
		return err
	} else if fs.NArg() > 0 {
		fs.Usage()
		return flag.ErrHelp
	}

	return nil
}

func (c *serve) run(ctx context.Context, stderr, stdout io.Writer) error {
	shudown, err := setupOTel(ctx)
	if err != nil {
		return err
	}
	defer shudown(ctx)

	pool, err := pgxpool.New(ctx, c.db.url)
	if err != nil {
		return err
	}
	defer pool.Close()

	return c.listenAndServe(ctx, pool)
}

func (c *serve) listenAndServe(ctx context.Context, pool *pgxpool.Pool) error {
	ctx, cancel := signal.NotifyContext(ctx, unix.SIGINT, unix.SIGKILL, unix.SIGTERM)
	defer cancel()

	s := &http.Server{
		Addr:        c.http.addr,
		Handler:     http.Handler(&o0.DB{RWC: pool}, &o1.DB{RWC: pool}, &o2.DB{RWC: pool}, &o3.DB{RWC: pool}),
		BaseContext: func(l net.Listener) context.Context { return ctx },
	}
	s.RegisterOnShutdown(cancel)

	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := s.ListenAndServe(); !http.IsServerClosed(err) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			return err
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}
