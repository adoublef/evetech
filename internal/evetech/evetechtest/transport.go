// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package evetechtest

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/adoublef/evetech/internal/evetech"
)

type Transporter struct{}

func (tt *Transporter) RoundTrip(req *http.Request) (*http.Response, error) {
	switch {
	case req.Method == http.MethodGet && req.URL.Path == "/v1/universe/regions":
		// ids endpoint
		ids := []int{10000002, 10000043}
		b, _ := json.Marshal(ids)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(b)),
			Header:     make(http.Header),
			Request:    req,
		}, nil

	case req.Method == http.MethodHead && strings.HasPrefix(req.URL.Path, "/v1/markets/") && strings.HasSuffix(req.URL.Path, "/orders"):
		// max endpoint
		h := make(http.Header)
		h.Set("x-pages", "5")
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(nil)),
			Header:     h,
			Request:    req,
		}, nil

	case req.Method == http.MethodGet && strings.HasPrefix(req.URL.Path, "/v1/markets/") && strings.HasSuffix(req.URL.Path, "/orders"):
		// orders endpoint
		orders := []evetech.Order{
			{OrderID: 1, Price: 123.45},
			{OrderID: 2, Price: 234.56},
		}
		b, _ := json.Marshal(orders)
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewReader(b)),
			Header:     make(http.Header),
			Request:    req,
		}, nil

	default:
		return &http.Response{
			StatusCode: 404,
			Body:       io.NopCloser(strings.NewReader("not found")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}
}
