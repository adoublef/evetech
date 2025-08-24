// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http_test

import (
	"net/http"
	"strings"
	"testing"

	"go.adoublef.dev/testing/is"
)

func Test_handleAdd(t *testing.T) {
	t.Run("OK", func(t *testing.T) {
		s, ctx := newServer(t)

		resp, err := post(ctx, s, "/v0/orders", strings.NewReader(`{"order_id":1}`), contentType("application/json"))
		is.OK(t, err) // POST /categories
		is.Equal(t, resp.StatusCode, http.StatusNoContent)
		close(t, resp.Body)
	})
}
