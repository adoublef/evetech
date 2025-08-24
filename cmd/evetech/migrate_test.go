// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"testing"

	"go.adoublef.dev/testing/is"
)

func Test_migrate(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		d, ctx := newDB(t)
		dsn := d.Config().ConnString()

		{ // up migration
			ctx, cancel := context.WithCancel(ctx)
			err := run(ctx, nil, nil, nil, nil, "migrate", "--database-url", dsn)
			is.OK(t, err)
			cancel()
		}
		{ // down migration
			ctx, cancel := context.WithCancel(ctx)
			err := run(ctx, nil, nil, nil, nil, "migrate", "--database-url", dsn, "--down")
			is.OK(t, err)
			cancel()
		}
	})
}
