package db

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/errwrap"
	"github.com/jmoiron/sqlx"
)

var db *sqlx.DB

// Conn is a handle for the database
func Conn() *sqlx.DB {
	return db
}

// Load DB configuration and ping the server.
func Load(ctx context.Context, dsn string) (*sqlx.DB, error) {
	var err error
	db, err = sqlx.Open("postgres", dsn)

	if err != nil {
		return nil, errwrap.Wrapf("can't open connection to database: {{err}}", err)
	}

	if err := db.PingContext(ctx); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "can't ping database: %v\n", err)
	}

	return db, nil
}
