// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package order

import (
	"context"

	"github.com/adoublef/evetech/internal/order"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.adoublef.dev/sync/batchque"
)

type Order = order.Order

type DB struct {
	RWC *pgxpool.Pool

	add batchque.Group[Order, struct{}]
}

func (d *DB) Orders(ctx context.Context, id int) ([]Order, error) {
	r, err := d.RWC.Query(ctx, `
		select row_to_json("order".*) 
		from "order"
		where order_id = $1
		limit 10; -- TODO
    `, id)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	oo := make([]Order, 0)
	for r.Next() {
		var o Order
		err := r.Scan(&o)
		if err != nil {
			return nil, err
		}
		oo = append(oo, o)
	}
	if err := r.Err(); err != nil {
		return nil, err
	}
	return oo, nil
}

// Add an [Order]
func (d *DB) Add(ctx context.Context, o Order) error {
	type Request = batchque.Request[Order, struct{}]
	_, err := d.add.Do(ctx, o, func(ctx context.Context, r []Request) {
		var pr = make([]Request, 0, len(r))
		var b = pgx.Batch{
			QueuedQueries: make([]*pgx.QueuedQuery, 0, len(r)),
		}
		for _, r := range r {
			select {
			case <-r.Context().Done():
				continue
			default:
				_ = b.Queue(`
					insert into "order"
					select *
					from json_populate_record(NULL::"order", $1::json);
				`, r.Val)
				pr = append(pr, r)
			}
		}
		br := d.RWC.SendBatch(ctx, &b)
		defer br.Close()

		for _, r := range pr {
			_, err := br.Exec()
			if err != nil {
				r.CancelFunc(err)
				continue
			}
			select {
			case <-r.Context().Done():
			default:
				close(r.C)
			}
		}
	})
	return err
}
