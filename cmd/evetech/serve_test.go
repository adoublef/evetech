// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"testing"
	"time"

	"go.adoublef.dev/testing/is"
	"go.adoublef.dev/testing/wait"
	"golang.org/x/sync/errgroup"
)

func Test_serve(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		d, ctx := newDB(t)
		dsn := d.Config().ConnString()

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		g, ctx := errgroup.WithContext(ctx)

		g.Go(func() error {
			return run(ctx, nil, nil, nil, nil, "serve", "--http-address", ":8888", "--database-url", dsn)
		})
		g.Go(func() error {
			defer cancel()
			return wait.ForFunc(ctx, 10*time.Second, func() error {
				return run(ctx, nil, nil, nil, nil, "health", "http://0.0.0.0:8888/ready")
			})
		})

		is.OK(t, g.Wait())
	})
}
