package auth

import (
	"context"
	"database/sql"

	"github.com/henvic/climetrics/db"
	"github.com/kisielk/sqlstruct"
)

// Authentication object
type Authentication struct {
	UserID   string `schema:"user_id"`
	Username string `schema:"username"`
	Email    string `schema:"email"`
	Role     string `schema:"role"`
	Password string `schema:"password"`
}

// Get user for authentication (not revoked)
func Get(ctx context.Context, email string) (a Authentication, err error) {
	var stmt *sql.Stmt

	conn := db.Conn()

	stmt, err = conn.PrepareContext(ctx,
		`SELECT user_id, username, email, password, role FROM authentication
		WHERE (email = $1 OR username = $1) AND role != $2 LIMIT 1`)

	if err != nil {
		return a, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	var rows *sql.Rows
	rows, err = stmt.QueryContext(ctx, email, "revoked")

	if err != nil {
		return a, err
	}

	if ok := rows.Next(); !ok {
		return a, sql.ErrNoRows
	}

	err = sqlstruct.Scan(&a, rows)
	return a, err
}
