package users

import (
	"context"
	"strings"

	"database/sql"

	"github.com/henvic/climetrics/db"
	"github.com/kisielk/sqlstruct"
)

// User structure
type User struct {
	UserID   string `schema:"user_id"`
	Username string `schema:"username"`
	Email    string `schema:"email"`
	Role     string `schema:"role"`
	Password string `schema:"password"`
}

// Filter for users.
type Filter struct {
	Active bool
}

// List users
func List(ctx context.Context, f Filter) (users []User, err error) {
	q := []string{"SELECT user_id, username, email, role FROM authentication"}
	args := []interface{}{}

	if f.Active {
		q = append(q, "WHERE")
		q = append(q, "role != $1")
		args = append(args, "revoked")
	}

	conn := db.Conn()
	stmt, err := conn.PrepareContext(ctx, strings.Join(q, " "))

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryContext(ctx, args...)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var u User
		err = sqlstruct.Scan(&u, rows)

		if err != nil {
			return nil, err
		}

		users = append(users, u)
	}

	return users, err
}

// Get gets an user row by ID
func Get(ctx context.Context, userID string) (u User, err error) {
	conn := db.Conn()
	stmt, err := conn.PrepareContext(ctx,
		`SELECT user_id, username, email, password, role FROM authentication WHERE user_id = $1`)

	if err != nil {
		return u, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryContext(ctx, userID)

	if err != nil {
		return u, err
	}

	if ok := rows.Next(); !ok {
		return u, sql.ErrNoRows
	}

	err = sqlstruct.Scan(&u, rows)
	return u, err
}

// Create user
func Create(ctx context.Context, user User) error {
	conn := db.Conn()
	stmt, err := conn.PrepareContext(ctx, `INSERT INTO authentication (user_id, username, email, password, role) VALUES ($1, $2, $3, $4, $5)`)

	if err != nil {
		return err
	}

	defer func() {
		_ = stmt.Close()
	}()

	_, err = stmt.ExecContext(ctx, user.UserID, user.Username, user.Email, user.Password, user.Role)
	return err
}

// Update user's data
func Update(ctx context.Context, u User) error {
	conn := db.Conn()
	stmt, err := conn.PrepareContext(ctx, "UPDATE authentication SET username = $1, email = $2, password = $3, role = $4 WHERE user_id = $5")

	if err != nil {
		return err
	}

	defer func() {
		_ = stmt.Close()
	}()

	_, err = stmt.ExecContext(ctx, u.Username, u.Email, u.Password, u.Role, u.UserID)
	return err
}
