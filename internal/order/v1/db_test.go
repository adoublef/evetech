// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package order_test

import (
	"testing"

	. "github.com/adoublef/evetech/internal/order/v1"
	"go.adoublef.dev/testing/is"
)

func TestDB_Add(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		d, ctx := newDB(t)

		o := Order{OrderID: 1, Price: 123.45}
		err := d.Add(ctx, o)
		is.OK(t, err) // DB.Add

		oo, err := d.Orders(ctx, 1)
		is.OK(t, err) // DB.Orders
		is.Equal(t, len(oo), 1)
		is.Equal(t, oo[0].Price, 123.45) // Order.Price
	})
}
