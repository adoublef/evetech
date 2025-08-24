// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os/signal"

	"golang.org/x/sys/unix"
)

type health struct {
	http struct {
		endpoint string
		code     int // --status
	}
}

func (c *health) parse(stdin io.Reader, getenv func(string) string, args ...string) error {
	fs := flag.NewFlagSet("health", flag.ContinueOnError)
	fs.IntVar(&c.http.code, "status", http.StatusOK, "status code")
	// todo: fs.Usage
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return flag.ErrHelp
	}
	c.http.endpoint = fs.Arg(0)
	return nil
}

func (c *health) run(ctx context.Context, stderr io.Writer, stdout io.Writer) error {
	ctx, cancel := signal.NotifyContext(ctx, unix.SIGINT, unix.SIGKILL, unix.SIGTERM)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.http.endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	// note: backoff would be useful, but assume that the environment will handle this for now.
	// note: config required, for example tls.
	res, err := (&http.Client{ /* conifg */ }).Do(req)
	if err != nil {
		return fmt.Errorf("failed making request: %w", err)
	}
	defer res.Body.Close()
	if n := res.StatusCode; n != c.http.code {
		return fmt.Errorf("service not ready: %d", n)
	}
	return nil
}
