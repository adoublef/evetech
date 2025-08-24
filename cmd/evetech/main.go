// Copyright 2025 Kristopher Rahim Afful-Brown. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

func main() {
	var (
		ctx    = context.Background()
		args   = os.Args[1:]
		getenv = os.Getenv
		stdin  = os.Stdin
		stderr = os.Stderr
		stdout = os.Stdout
	)

	err := run(ctx, stderr, stdout, stdin, getenv, args...)
	if errors.Is(err, flag.ErrHelp) {
		os.Exit(2)
	} else if err != nil {
		fmt.Fprintf(stderr, "[ERR]: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, stderr, stdout io.Writer, stdin io.Reader, getenv func(string) string, args ...string) error {
	var name string
	if len(args) > 0 {
		name, args = args[0], args[1:]
	}
	type cmd interface {
		parse(stdin io.Reader, getenv func(string) string, args ...string) error
		run(ctx context.Context, stderr, stdout io.Writer) error
	}
	for s, c := range map[string]cmd{"serve": &serve{}, "health": &health{}, "migrate": &migrate{}} {
		if s == name {
			if err := c.parse(stdin, getenv, args...); err != nil {
				return err
			}
			return c.run(ctx, stderr, stdout)
		}
	}
	return fmt.Errorf("command %s not found", name)
}
