// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/adoublef/evetech/internal/evetech/evetechtest"
	"go.adoublef.dev/testing/is"
)

func Test_do(t *testing.T) {
	ctx := t.Context()
	w := io.Discard
	c := &http.Client{
		Transport: &evetechtest.Transporter{},
	}

	n, err := do(ctx, c, w)
	is.OK(t, err) // do

	/*
		1. ids returns 2 region IDs
		1. for each id, max returns 5.
		1. for each page, orders returns 2.

		total: 1 (header) + 20 (orders) = 21
	*/
	is.Equal(t, n, 21) // header
}
