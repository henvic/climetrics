package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/henvic/climetrics/db"
	"github.com/henvic/climetrics/metrics"
	_ "github.com/lib/pq"
)

var dsn string

func setup(ctx context.Context) error {
	_, err := db.Load(ctx, dsn)
	return err
}

func run() error {
	rand.Seed(time.Now().UTC().UnixNano())
	flag.Parse()

	ctx := context.Background()

	if err := setup(ctx); err != nil {
		return err
	}

	missing, err := metrics.MissingGeolocation(ctx)

	if err != nil {
		return err
	}

	return add(ctx, missing)
}

func add(ctx context.Context, ips []string) error {
	var errored []string

	for _, ip := range ips {
		updated, err := metrics.AddGeolocationIP(ctx, ip)

		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "cannot find geolocation for IP %s: %v\n", ip, err)
			errored = append(errored, ip)
			continue
		}

		fmt.Printf("%d entries updated with IP %s\n", updated, ip)
	}

	if len(errored) != 0 {
		return fmt.Errorf("failed to gather geolocation information for %d IPs", len(errored))
	}

	return nil
}

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&dsn, "dsn", "postgres://admin@/climetrics?sslmode=disable", "dsn (PostgreSQL)")
}
