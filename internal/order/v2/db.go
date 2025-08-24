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
        select
            duration,
            is_buy_order,
            issued,
            location_id,
            min_volume,
            order_id,
            price,
            range,
            system_id,
            type_id,
            volume_remain,
            volume_total
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
		err := r.Scan(
			&o.Duration,
			&o.IsBuyOrder,
			&o.Issued,
			&o.LocationID,
			&o.MinVolume,
			&o.OrderID,
			&o.Price,
			&o.Range,
			&o.SystemID,
			&o.TypeID,
			&o.VolumeRemain,
			&o.VolumeTotal,
		)
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
				o := r.Val
				_ = b.Queue(`
					insert into "order" (
						duration, is_buy_order, issued, location_id, min_volume, order_id,
						price, range, system_id, type_id, volume_remain, volume_total
					) values (
						$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
					);
				`, o.Duration,
					o.IsBuyOrder,
					o.Issued,
					o.LocationID,
					o.MinVolume,
					o.OrderID,
					o.Price,
					o.Range,
					o.SystemID,
					o.TypeID,
					o.VolumeRemain,
					o.VolumeTotal)
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
