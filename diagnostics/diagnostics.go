package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/hashicorp/errwrap"
	"github.com/henvic/climetrics/db"
	"github.com/henvic/climetrics/timejson"
	"github.com/kisielk/sqlstruct"
	"github.com/satori/go.uuid"
)

// Report structure for the diagnostics
type Report struct {
	ID        string `db:"id" json:"id,omitempty"`
	Username  string `db:"username" json:"username,omitempty"`
	Report    string `db:"report" json:"report,omitempty"`
	Timestamp string `db:"timestamp" json:"time,omitempty"`
	SyncTime  string `db:"sync_time" json:"sync_time,omitempty"`

	TimestampDB timejson.RubyDate `db:"timestamp_db"`
}

// HumanSyncTime returns a human-readable SyncTime.
func (r Report) HumanSyncTime() string {
	return humanTime(time.RFC3339, r.SyncTime)
}

// HumanTimestamp returns a human-readable Timestamp.
func (r Report) HumanTimestamp() string {
	return fmt.Sprintf("%s (%s)", r.Timestamp, humanize.Time(time.Time(r.TimestampDB)))
}

func humanTime(layout, value string) string {
	t, err := time.Parse(layout, value)

	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("%s (%s)", t.Format(time.UnixDate), humanize.Time(t))
}

// Create report
func Create(ctx context.Context, r Report) (err error) {
	r.Username = strings.ToLower(r.Username)

	// check if report.ID is on the RFC4122 version 4 format with no urn prefix:
	if strings.HasPrefix(r.ID, "urn:") {
		return errors.New("expected no urn: on report ID")
	}

	var u uuid.UUID
	u, err = uuid.FromString(r.ID)

	if err != nil || u.Version() != 4 || u.Variant() != uuid.VariantRFC4122 {
		return errors.New("invalid UUID")
	}

	r.ID = strings.ToLower(u.String())

	ts, err := time.Parse(time.RubyDate, r.Timestamp)

	if err != nil {
		return errwrap.Wrapf("invalid diagnostics timestamp: {{err}}", err)
	}

	r.TimestampDB = timejson.RubyDate(ts)

	conn := db.Conn()

	stmt, err := conn.PreparexContext(ctx,
		`INSERT INTO diagnostics
("id", "username", "report", "timestamp", "timestamp_db")
VALUES ($1, $2, $3, $4, $5)
`)

	if err != nil {
		return err
	}

	defer func() {
		_ = stmt.Close()
	}()

	_, err = stmt.ExecContext(ctx, r.ID, r.Username, r.Report, r.Timestamp, r.TimestampDB)
	return err
}

// Filter sets the filter settings
type Filter struct {
	Username         string
	UsernameOperator Op

	Page    int
	PerPage int
}

// Op for filtering.
type Op string

const (
	// Contains operator.
	Contains Op = "contains"

	// Like operator.
	Like Op = "like"

	// Equal operator.
	Equal Op = "equal"
)

// Count reports.
func Count(ctx context.Context, f Filter) (int, error) {
	var q = []string{"SELECT COUNT(id) FROM diagnostics"}
	var args = []interface{}{}

	if f.Username != "" {
		q = append(q, "WHERE")

		switch f.UsernameOperator {
		case Contains:
			q = append(q, "username ILIKE $1")
			args = append(args, `%`+f.Username+`%`)
		case Like:
			q = append(q, "username ILIKE $1")
			args = append(args, f.Username)
		case Equal:
			q = append(q, "username = $1")
			args = append(args, f.Username)
		default:
			return 0, fmt.Errorf(`invalid username equality operator "%s"`, f.UsernameOperator)
		}
	}

	conn := db.Conn()
	stmt, err := conn.PreparexContext(ctx, strings.Join(q, " "))

	if err != nil {
		return 0, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryxContext(ctx, args...)

	if err != nil {
		return 0, err
	}

	var count = 0

	if rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, err
		}
	}

	return count, nil
}

// List diagnostics
func List(ctx context.Context, f Filter) (reports []Report, err error) {
	var q = []string{
		"SELECT id, username, timestamp, timestamp_db, sync_time FROM diagnostics",
	}

	if f.Page == 0 {
		f.Page = 1
	}

	var args = []interface{}{
		f.PerPage,
		(f.Page - 1) * f.PerPage,
	}

	if f.Username != "" {
		q = append(q, "WHERE")

		switch f.UsernameOperator {
		case Contains:
			q = append(q, "username ILIKE $3")
			args = append(args, `%`+f.Username+`%`)
		case Like:
			q = append(q, "username ILIKE $3")
			args = append(args, f.Username)
		case Equal:
			q = append(q, "username = $3")
			args = append(args, f.Username)
		default:
			return nil, fmt.Errorf(`invalid username equality operator "%s"`, f.UsernameOperator)
		}
	}

	q = append(q, "ORDER BY sync_time DESC LIMIT $1 OFFSET $2")

	conn := db.Conn()
	stmt, err := conn.PreparexContext(ctx, strings.Join(q, " "))

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryxContext(ctx, args...)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var r Report
		err = sqlstruct.Scan(&r, rows)

		if err != nil {
			return nil, err
		}

		reports = append(reports, r)
	}

	return reports, err
}

// Get diagnostics report
func Get(ctx context.Context, id string) (r Report, err error) {
	var q = "SELECT id, username, report, timestamp, timestamp_db, sync_time FROM diagnostics WHERE id = $1"

	conn := db.Conn()
	stmt, err := conn.PreparexContext(ctx, q)

	if err != nil {
		return r, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	row := stmt.QueryRowxContext(ctx, id)

	if err = row.Err(); err != nil {
		return r, err
	}

	err = row.StructScan(&r)
	return r, err
}
