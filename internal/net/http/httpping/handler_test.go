// Copyright 2025 The Mosaic Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpping_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	. "github.com/adoublef/evetech/internal/net/http/httpping"
	"go.adoublef.dev/testing/is"
)

func TestHandler(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		get := newGet(t, PingerFunc(func(ctx context.Context) error { return nil }))

		resp := get(t.Context())
		is.Equal(t, resp.StatusCode, http.StatusOK)
	})

	t.Run("Sequential", func(t *testing.T) {
		var count int64
		get := newGet(t, PingerFunc(func(ctx context.Context) error { atomic.AddInt64(&count, 1); return nil }))

		for range 8 {
			_ = get(t.Context()).Body.Close()
		}

		is.Equal(t, count, 1)
	})

	t.Run("Concurrent", func(t *testing.T) {
		var count int64
		get := newGet(t, PingerFunc(func(ctx context.Context) error { atomic.AddInt64(&count, 1); return nil }))

		var wg sync.WaitGroup
		for range 8 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = get(t.Context()).Body.Close()
			}()
		}
		wg.Wait()

		is.Equal(t, count, 1)
	})
}

func newGet(t testing.TB, p Pinger) func(context.Context) *http.Response {
	t.Helper()

	s := newServer(t, p)
	t.Cleanup(func() { s.Close() })

	get := func(ctx context.Context) *http.Response {
		t.Helper()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL, nil)
		is.OK(t, err) // http.NewRequestWithContext

		res, err := s.Client().Do(req)
		is.OK(t, err) // http.Client.Do

		return res
	}
	return get
}

func newServer(t testing.TB, p Pinger) *httptest.Server {
	t.Helper()
	return httptest.NewServer(Handler(p, 0))
}
