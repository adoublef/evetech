// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http

import (
	"context"
	"net/http"

	"github.com/adoublef/evetech/internal/order"
	"go.adoublef.dev/net/xhttp"
	olog "go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

const scopeName = "github.com/adoublef/evetech/internal/net/http"

var (
	tracer = otel.Tracer(scopeName)
	_      = otel.Meter(scopeName)
	_      = olog.NewLogger(scopeName)
)

type DB interface {
	Add(ctx context.Context, o order.Order) error
	Orders(ctx context.Context, id int) ([]order.Order, error)
}

func Handler(d0, d1, d2, d3 DB) http.Handler {
	mux := http.NewServeMux()
	handleFunc := func(pattern string, h http.Handler) {
		h = otelhttp.WithRouteTag(pattern, h)
		mux.Handle(pattern, h)
	}

	handleFunc("/v0/orders", handleAdd(d0, "v0"))
	handleFunc("/v1/orders", handleAdd(d1, "v1"))
	handleFunc("/v2/orders", handleAdd(d2, "v2"))
	handleFunc("/v3/orders", handleAdd(d3, "v3"))

	h := otelhttp.NewHandler(mux, "/")
	return h
}

func handleAdd(d DB, span string) http.HandlerFunc {
	span = "handleAdd" + span
	parse := func(w http.ResponseWriter, r *http.Request) (order.Order, error) {
		return xhttp.Decode[order.Order](w, r, 0, 0)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := tracer.Start(r.Context(), span)
		defer span.End()

		o, err := parse(w, r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := d.Add(ctx, o); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
