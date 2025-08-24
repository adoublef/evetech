package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	. "github.com/adoublef/evetech/internal/database/postgres"
	"github.com/testcontainers/testcontainers-go"
	"go.adoublef.dev/runtime/container/postgres"
	"go.adoublef.dev/testing/is"
)

func TestUp(t *testing.T) {
	ctx := t.Context()

	p, err := container.ConnectionPool(ctx)
	is.OK(t, err) // container.ConnectionPool
	t.Cleanup(func() { p.Close() })

	dsn := p.Config().ConnString()

	is.OK(t, Up(ctx, dsn))   // Up
	is.OK(t, Down(ctx, dsn)) // Down
}

func TestMain(m *testing.M) {
	err := setup(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	code := m.Run()
	err = cleanup(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	os.Exit(code)
}

var container *postgres.Container

// setup initialises containers within the pacakge.
func setup(ctx context.Context) (err error) {
	container, err = postgres.Run(ctx, "") // 17-alpine
	if err != nil {
		return
	}
	return
}

// cleanup stops all running containers for the pacakge.
func cleanup(ctx context.Context) (err error) {
	var cc = []testcontainers.Container{container}
	for _, c := range cc {
		if c != nil {
			err = errors.Join(err, c.Terminate(ctx))
		}
	}
	return err
}
