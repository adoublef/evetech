package main

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"iter"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"

	"github.com/adoublef/evetech/internal/order"
	"golang.org/x/sync/errgroup"
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
		args   = os.Args[1:]
	)

	err := run(ctx, args, getenv, stdin, stderr, stdout)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(2)
	} else if err != nil {
		fmt.Fprintf(stderr, "ERR: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, _ func(string) string, _ io.Reader, stderr, _ io.Writer) error {
	if len(args) < 1 || args[0] == "" {
		return fmt.Errorf("missing endpoint argument")
	}
	endpoint := args[0]

	ctx, cancel := signal.NotifyContext(ctx, unix.SIGINT, unix.SIGKILL, unix.SIGTERM)
	defer cancel()

	written, err := do(ctx, &http.Client{}, endpoint)
	fmt.Fprintf(stderr, "written %d to database\n", written)
	return err
}

func do(ctx context.Context, c *http.Client, endpoint string) (int, error) {
	if c == nil {
		c = http.DefaultClient
	}
	// c.Timeout = 15 * time.Second
	// if t, ok := c.Transport.(*http.Transport); ok {
	// 	t.MaxIdleConns = 100
	// 	t.MaxIdleConnsPerHost = 10
	// 	t.IdleConnTimeout = 90 * time.Second
	// 	t.DialContext = (&net.Dialer{
	// 		Timeout:   10 * time.Second,
	// 		KeepAlive: 30 * time.Second,
	// 	}).DialContext
	// 	t.TLSHandshakeTimeout = 10 * time.Second
	// 	t.ResponseHeaderTimeout = 10 * time.Second
	// 	t.ExpectContinueTimeout = 1 * time.Second
	// }

	queries, g1 := queries(ctx, c, 10)
	entries, g2 := entries(ctx, c, queries, 50)
	written, g3 := collect(ctx, c, endpoint, entries, 40)
	// ERR: Post "http://evetech.localhost/v2/orders": dial tcp 127.0.0.1:80: connect: can't assign requested address
	if err := cmp.Or(g1.Wait(), g2.Wait(), g3.Wait()); err != nil {
		return int(written), err
	}
	return int(written), nil
}

func collect(ctx context.Context, c *http.Client, endpoint string, entries <-chan order.Order, limit int) (int64, interface{ Wait() error }) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	var written int64
	for entry := range entries {
		g.Go(func() error {
			err := post(ctx, c, endpoint, entry)
			if err == nil {
				atomic.AddInt64(&written, 1)
			}
			return err
		})
	}
	return written, g
}

func ready(ctx context.Context, c *http.Client, endpoint string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	default:
		p, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to write order: %s", string(p))
	}
}

func post(ctx context.Context, c *http.Client, endpoint string, o order.Order) error {
	// pr, pw := io.Pipe()
	// defer pr.Close()

	// go func() {
	// 	defer pw.Close()
	// 	if err := json.NewEncoder(pw).Encode(o); err != nil {
	// 		pw.CloseWithError(err)
	// 	}
	// }()

	p, err := json.Marshal(o)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(p))
	if err != nil {
		return err
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil
	default:
		p, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to write order: %s", string(p))
	}
}

type query struct {
	id, page int
}

func queries(ctx context.Context, c *http.Client, limit int) (<-chan query, interface{ Wait() error }) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	queries := make(chan query, 1)
	go func() {
		for id, err := range ids(ctx, c) {
			g.Go(func() error {
				if err != nil {
					return err
				}
				max, err := max(ctx, c, id)
				if err != nil {
					return err
				}
				for i := range max {
					select {
					case queries <- query{id, i + 1}:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
				return nil
			})
		}
		g.Wait()
		close(queries)
	}()
	return queries, g
}

func entries(ctx context.Context, c *http.Client, queries <-chan query, limit int) (<-chan order.Order, interface{ Wait() error }) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(limit)
	ch := make(chan order.Order, 1)
	go func() {
		for q := range queries {
			g.Go(func() error {
				for order, err := range orders(ctx, c, q.id, q.page) {
					if err != nil {
						return err
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case ch <- order:
					}
				}
				return nil
			})
		}
		g.Wait()
		close(ch)
	}()
	return ch, g
}

func ids(ctx context.Context, c *http.Client) iter.Seq2[int, error] {
	url := "https://esi.evetech.net/v1/universe/regions"
	return func(yield func(int, error) bool) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil && !yield(0, err) {
			return
		}
		req.Close = true
		resp, err := c.Do(req)
		if err != nil && !yield(0, err) {
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode > 300 {
			p, _ := io.ReadAll(resp.Body)
			if !yield(0, fmt.Errorf("ids: %s", string(p))) {
				return
			}
		}

		d := json.NewDecoder(resp.Body)
		if _, err := d.Token(); err != nil && !yield(0, err) {
			return
		}
		for d.More() {
			var n int
			if err := d.Decode(&n); !yield(n, err) {
				return
			}
		}
		if _, err = d.Token(); err != nil && !yield(0, err) {
			return
		}
	}
}

func max(ctx context.Context, c *http.Client, id int) (int, error) {
	url := fmt.Sprintf("https://esi.evetech.net/v1/markets/%d/orders", id)
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil) // context
	if err != nil {
		return 0, err
	}
	req.Close = true
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

func orders(ctx context.Context, c *http.Client, id, page int) iter.Seq2[order.Order, error] {
	url := fmt.Sprintf("https://esi.evetech.net/v1/markets/%d/orders?page=%d", id, page)
	return func(yield func(order.Order, error) bool) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) // context
		if err != nil && !yield(order.Order{}, err) {
			return
		}
		req.Close = true
		resp, err := c.Do(req)
		if err != nil && !yield(order.Order{}, err) {
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode > 300 {
			p, _ := io.ReadAll(resp.Body)
			if !yield(order.Order{}, fmt.Errorf("orders: %s", string(p))) {
				return
			}
		}

		d := json.NewDecoder(resp.Body)
		if _, err := d.Token(); err != nil && !yield(order.Order{}, err) {
			return
		}
		for d.More() {
			var o order.Order
			if err := d.Decode(&o); !yield(o, err) {
				return
			}
		}
		if _, err = d.Token(); err != nil && !yield(order.Order{}, err) {
			return
		}
	}
}
