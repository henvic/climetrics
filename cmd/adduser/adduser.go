package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"syscall"
	"time"

	"github.com/henvic/climetrics/db"
	_ "github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

var dsn string

func prompt() (string, error) {
	var scanner = bufio.NewScanner(os.Stdin)

	if scanner.Scan() {
		return scanner.Text(), nil
	}

	return "", scanner.Err()
}

func setup(ctx context.Context) error {
	return db.Load(ctx, dsn)
}

type user struct {
	Username string
	Email    string
	Password string
	Role     string
}

func run() error {
	rand.Seed(time.Now().UTC().UnixNano())
	flag.Parse()

	ctx := context.Background()

	var err error

	if err = setup(ctx); err != nil {
		return err
	}

	fmt.Println("Adding user to CLI Metrics")

	var u user

	fmt.Print("username: ")
	if u.Username, err = prompt(); err != nil {
		return err
	}

	fmt.Print("email: ")
	if u.Email, err = prompt(); err != nil {
		return err
	}

	fmt.Print("Password: ")
	if u.Password, err = getPassword(); err != nil {
		return err
	}
	fmt.Println("█")

	fmt.Print("Access [admin/member] (default: member): ")
	if u.Role, err = prompt(); err != nil {
		return err
	}

	if u.Role == "" {
		u.Role = "member"
	}

	return add(ctx, u)
}

func add(ctx context.Context, u user) error {
	conn := db.Conn()

	stmt, err := conn.PreparexContext(ctx, `INSERT INTO authentication
	("user_id", "username", "email", "password", "role")
	VALUES ($1, $2, $3, $4, $5)`)

	if err != nil {
		return err
	}

	defer func() {
		_ = stmt.Close()
	}()

	uid := uuid.NewV4().String()

	_, err = stmt.ExecContext(ctx, uid, u.Username, u.Email, u.Password, u.Role)

	if err == nil {
		fmt.Printf("\nUser \"%s\" created.\n", uid)
	}

	return err
}

func readPassword() (string, error) {
	b, err := terminal.ReadPassword(int(syscall.Stdin))
	return string(b), err
}

func getPassword() (string, error) {
	password, err := readPassword()

	if err != nil {
		return "", err
	}

	var hash []byte
	hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
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
