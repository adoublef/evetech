// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/adoublef/evetech/internal/database/postgres"
	. "github.com/adoublef/evetech/internal/net/http"
	o0 "github.com/adoublef/evetech/internal/order"
	o1 "github.com/adoublef/evetech/internal/order/v1"
	o2 "github.com/adoublef/evetech/internal/order/v2"
	o3 "github.com/adoublef/evetech/internal/order/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	postgresc "go.adoublef.dev/runtime/container/postgres"
	"go.adoublef.dev/testing/is"
	"golang.org/x/sync/errgroup"
)

func newServer(t testing.TB) (*httptest.Server, context.Context) {
	t.Helper()

	ctx := t.Context()

	p := newPool(t)

	s := httptest.NewServer(Handler(&o0.DB{RWC: p}, &o1.DB{RWC: p}, &o2.DB{RWC: p}, &o3.DB{RWC: p}))
	t.Cleanup(func() { s.Close() })

	return s, ctx
}

func contentType(s string) func(r *http.Request) {
	return func(r *http.Request) {
		r.Header.Add("Content-Type", s)
	}
}

func get(ctx context.Context, s *httptest.Server, path string, opts ...func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL+path, nil)
	if err != nil {
		return nil, err
	}
	for _, f := range opts {
		f(req)
	}
	return s.Client().Do(req)
}

func post(ctx context.Context, s *httptest.Server, path string, body io.Reader, opts ...func(*http.Request)) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.URL+path, body)
	if err != nil {
		return nil, err
	}
	for _, f := range opts {
		f(req)
	}
	return s.Client().Do(req)
}

func decode[V any](t testing.TB, r io.ReadCloser) V {
	t.Helper()

	var v V
	err := json.NewDecoder(r).Decode(&v)
	is.OK(t, err)
	close(t, r)

	return v
}

func close(t testing.TB, closer io.Closer) {
	t.Helper()
	is.OK(t, closer.Close())
}

func newPool(t testing.TB) *pgxpool.Pool {
	t.Helper()
	ctx := t.Context()

	p, err := postgresContainer.ConnectionPool(ctx)
	is.OK(t, err) // postgresContainer.ConnectionPool(ctx)
	t.Cleanup(func() { p.Close() })

	dsn := p.Config().ConnString()
	err = postgres.Up(ctx, dsn)
	is.OK(t, err) // timescale.Up
	t.Cleanup(func() { is.OK(t, postgres.Down(context.Background(), dsn)) /* down */ })

	return p
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
