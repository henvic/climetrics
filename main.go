package main

import (
	"context"
	_ "expvar"
	"flag"
	"math/rand"
	"time"

	_ "github.com/henvic/climetrics/modules"
	"github.com/henvic/climetrics/server"
	"github.com/henvic/ctxsignal"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

var params = server.Params{}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	flag.Parse()

	ctx, cancel := ctxsignal.WithTermination(context.Background())
	defer cancel()

	if err := server.Start(ctx, params); err != nil {
		log.Fatal(err)
	}
}

func init() {
	flag.StringVar(&params.Address, "addr", "127.0.0.1:8080", "Serving address")
	flag.StringVar(&params.DSN, "dsn", "postgres://admin@/climetrics?sslmode=disable", "dsn (PostgreSQL)")
}
