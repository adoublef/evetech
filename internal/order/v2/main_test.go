// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package order_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/adoublef/evetech/internal/database/postgres"
	. "github.com/adoublef/evetech/internal/order/v2"
	"github.com/testcontainers/testcontainers-go"
	postgresc "go.adoublef.dev/runtime/container/postgres"
	"go.adoublef.dev/testing/is"
	"golang.org/x/sync/errgroup"
)

func newDB(t testing.TB) (*DB, context.Context) {
	t.Helper()
	ctx := t.Context()

	p, err := postgresContainer.ConnectionPool(ctx)
	is.OK(t, err) // postgresContainer.ConnectionPool(ctx)
	t.Cleanup(func() { p.Close() })

	dsn := p.Config().ConnString()
	err = postgres.Up(ctx, dsn)
	is.OK(t, err) // timescale.Up
	t.Cleanup(func() { is.OK(t, postgres.Down(context.Background(), dsn)) /* down */ })

	return &DB{RWC: p}, ctx
}

func TestMain(m *testing.M) {
	err := setup(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	err = cleanup(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

var postgresContainer *postgresc.Container

// setup initialises containers within the pacakge.
func setup(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() (err error) {
		postgresContainer, err = postgresc.Run(ctx, "")
		return
	})
	return g.Wait()
}

// cleanup stops all running containers for the pacakge.
func cleanup(ctx context.Context) (err error) {
	g := new(errgroup.Group)
	var cc = []testcontainers.Container{postgresContainer}
	for _, c := range cc {
		// if c != nil {
		g.Go(func() error { return c.Terminate(ctx) })
		// }
	}
	return g.Wait()
}
