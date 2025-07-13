// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"

	"github.com/adoublef/evetech/internal/evetech"
	"go.adoublef.dev/runtime/xprof"
	"golang.org/x/sys/unix"
)

// go tool pprof -http=:6060 cpu.pprof
// go tool trace trace.out
func main() {
	var (
		ctx    = context.Background()
		getenv = os.Getenv
		stdin  = os.Stdin
		stderr = os.Stderr
		stdout = os.Stdout
	)

	mode := flag.String("mode", "", xprof.Usage)
	q := flag.Bool("q", false, "quite mode")
	flag.Parse()

	args := flag.Args()

	var (
		opts []func(*xprof.Prof)
	)
	switch *mode {
	case "cpu":
		opts = append(opts, xprof.CPU)
	case "heap": // To find memory leaks or high memory usage
		opts = append(opts, xprof.Mem)
	case "alloc": // To optimize allocation rates (reduce GC pressure)
		opts = append(opts, xprof.MemAllocs)
	case "trace":
		opts = append(opts, xprof.Trace)
	}
	if len(opts) > 0 {
		if *q {
			opts = append(opts, xprof.Quiet)
		}
		defer xprof.Start("bench/v2", opts...).Stop()
	}

	err := run(ctx, args, getenv, stdin, stderr, stdout)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(2)
	} else if err != nil {
		fmt.Fprintf(stderr, "ERR: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, _ []string, _ func(string) string, _ io.Reader, _, _ io.Writer) error {
	ctx, cancel := signal.NotifyContext(ctx, unix.SIGINT, unix.SIGKILL, unix.SIGTERM)
	defer cancel()

	f, err := os.Create("results.csv")
	if err != nil {
		return fmt.Errorf("failed to create: %w", err)
	}
	defer f.Close()

	_, err = do(ctx, &http.Client{}, f)
	return err
}

func do(ctx context.Context, c *http.Client, w io.Writer) (written int, err error) {
	// httpClient := c
	if c == nil {
		c = http.DefaultClient
	}

	cw := csv.NewWriter(w)
	header := []string{
		"duration",
		"is_buy_order",
		"issued",
		"location_id",
		"min_volume",
		"order_id",
		"price",
		"range",
		"system_id",
		"type_id",
		"volume_remain",
		"volume_total",
	}
	if err := cw.Write(header); err != nil {
		return 0, fmt.Errorf("writing header: %w", err)
	}
	written++

	ids, err := ids(ctx, c)
	if err != nil {
		return written, err
	}

	for _, id := range ids {
		n, err := max(ctx, c, id)
		if err != nil {
			return written, err
		}
		for i := range n {
			orders, err := orders(ctx, c, id, i+1)
			if err != nil {
				return written, err
			}
			for _, order := range orders {
				if err := cw.Write(order.Record()); err != nil {
					return written, err
				}
				written++
			}
		}
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return 0, err
	}
	return written, nil
}

func ids(ctx context.Context, c *http.Client) ([]int, error) {
	url := "https://esi.evetech.net/v1/universe/regions"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		p, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ids: %s", string(p))
	}

	var ids []int
	err = json.NewDecoder(resp.Body).Decode(&ids)
	return ids, nil
}

func max(ctx context.Context, c *http.Client, id int) (int, error) {
	url := fmt.Sprintf("https://esi.evetech.net/v1/markets/%d/orders", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil) // context
	if err != nil {
		return 0, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode > 300 {
		p, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("max: %s", string(p))
	}
	return strconv.Atoi(resp.Header.Get("x-pages"))
}

func orders(ctx context.Context, c *http.Client, id, page int) ([]evetech.Order, error) {
	url := fmt.Sprintf("https://esi.evetech.net/v1/markets/%d/orders?page=%d", id, page)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode > 300 {
		p, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ids: %s", string(p))
	}

	var orders []evetech.Order
	err = json.NewDecoder(resp.Body).Decode(&orders)
	return orders, nil
}
