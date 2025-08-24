// Copyright 2025 The Mosaic Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package httpping

import (
	"context"
	"net/http"
	"sync"
	"time"

	"golang.org/x/sync/semaphore"
)

// Pinger
type Pinger interface {
	Ping(context.Context) error
}

// PingerFunc
type PingerFunc func(context.Context) error

func (f PingerFunc) Ping(ctx context.Context) error { return f(ctx) }

// Handler
func Handler(p Pinger, ttl time.Duration) http.Handler {
	if ttl == 0 {
		ttl = 60 * time.Second
	}
	sem := semaphore.NewWeighted(1)
	type cached struct {
		err error
		exp time.Time
	}
	var mu sync.RWMutex
	cache := make(map[struct{}]cached)

	ping := func(ctx context.Context) error {
		mu.RLock()
		if c, ok := cache[struct{}{}]; ok {
			if time.Now().After(c.exp) && sem.TryAcquire(1) {
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()

					mu.Lock()
					err := p.Ping(ctx)
					cache[struct{}{}] = cached{err, time.Now().Add(ttl)}
					mu.Unlock()
				}()
				mu.RUnlock()
				return c.err
			}
			mu.RUnlock()
			return c.err
		}
		mu.RUnlock()

		mu.Lock()
		err := p.Ping(ctx)
		cache[struct{}{}] = cached{err, time.Now().Add(ttl)}
		mu.Unlock()
		return err
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := ping(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
