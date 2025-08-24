// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package order

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	RWC *pgxpool.Pool
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
	_, err := d.RWC.Exec(ctx, `
        insert into "order" (
            duration, is_buy_order, issued, location_id, min_volume, order_id,
            price, range, system_id, type_id, volume_remain, volume_total
        ) values (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
        );
    `,
		o.Duration,
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
		o.VolumeTotal,
	)
	return err
}
