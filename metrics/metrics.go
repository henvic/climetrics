package metrics

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/hashicorp/errwrap"
	"github.com/henvic/climetrics/countrycode"
	"github.com/henvic/climetrics/db"
	"github.com/henvic/climetrics/geolocation"
	"github.com/henvic/climetrics/timejson"
	"github.com/kisielk/sqlstruct"
	uuid "github.com/satori/go.uuid"
)

// Metric entry.
type Metric struct {
	ID           string    `db:"id" json:"id"`
	Type         string    `db:"type" json:"event_type,omitempty"`
	Text         string    `db:"text" json:"text,omitempty"`
	Tags         Tags      `db:"tags" json:"tags,omitempty"`
	Extra        Extra     `db:"extra" json:"extra,omitempty"`
	PID          string    `db:"pid" json:"pid,omitempty"`
	SID          string    `db:"sid" json:"sid,omitempty"`
	Timestamp    string    `db:"timestamp" json:"time,omitempty"`
	Version      string    `db:"version" json:"version,omitempty"`
	OS           string    `db:"os" json:"os,omitempty"`
	Arch         string    `db:"arch" json:"arch,omitempty"`
	SyncTime     string    `db:"sync_time" json:"sync_time,omitempty"`
	RequestID    string    `db:"request_id" json:"request_id,omitempty"`
	SyncIP       string    `db:"sync_ip" json:"sync_ip,omitempty"`
	SyncLocation *Location `db:"sync_location" json:"sync_location,omitempty"`

	TimestampDB timejson.RubyDate `db:"timestamp_db"`
}

// HumanSyncTime returns a human-readable SyncTime.
func (m Metric) HumanSyncTime() string {
	return humanTime(time.RFC3339, m.SyncTime)
}

// HumanTimestamp returns a human-readable Timestamp.
func (m Metric) HumanTimestamp() string {
	return fmt.Sprintf("%s (%s)", m.Timestamp, humanize.Time(time.Time(m.TimestampDB)))
}

func humanTime(layout, value string) string {
	t, err := time.Parse(layout, value)

	if err != nil {
		return err.Error()
	}

	return fmt.Sprintf("%s (%s)", t.Format(time.UnixDate), humanize.Time(t))
}

// Location for the metric.
type Location struct {
	IP           string `json:"ip,omitempty"`
	City         string `json:"city,omitempty"`
	Region       string `json:"region,omitempty"`
	Country      string `json:"country,omitempty"`
	Coordinates  string `json:"loc,omitempty"`
	Organization string `json:"org,omitempty"`
	Bogon        bool   `json:"bogon,omitempty"`
	Cached       string `json:"cached,omitempty"`

	Error     map[string]string `json:"error,omitempty"`
	ErrorJSON error             `json:"error_json,omitempty"`
}

// Address (human readable) for the location.
func (l Location) Address() string {
	if l.Error != nil {
		return ""
	}

	var addr []string

	if l.City != "" {
		addr = append(addr, l.City)
	}

	if l.Region != "" && l.City != l.Region {
		addr = append(addr, l.Region)
	}

	if l.Country != "" {
		addr = append(addr, countrycode2name(l.Country))
	}

	return strings.Join(addr, ", ")
}

// Scan implements the Scanner interface.
func (l *Location) Scan(value interface{}) error {
	if err := json.Unmarshal(value.([]byte), &l); err != nil {
		l.ErrorJSON = err
	}

	return nil
}

// Value implements the driver Valuer interface.
func (l Location) Value() (driver.Value, error) {
	return json.Marshal(l)
}

func countrycode2name(code string) string {
	if name := countrycode.Get(code); name != "" {
		return name
	}

	return code
}

// Tags structure.
type Tags []string

// Scan implements the Scanner interface.
func (t *Tags) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &t)
}

// Value implements the driver Valuer interface.
func (t Tags) Value() (driver.Value, error) {
	return json.Marshal(t)
}

// Extra information structure.
type Extra map[string]string

// Scan implements the Scanner interface.
func (e *Extra) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &e)
}

// Value implements the driver Valuer interface.
func (e Extra) Value() (driver.Value, error) {
	return json.Marshal(e)
}

// Create report
func Create(ctx context.Context, m Metric) (created bool, err error) {
	if m.Type == "command_exec" {
		m.Type = "cmd"
	}

	if m.Type == "required_auth_cmd_precondition_failure" {
		m.Type = "required_auth"
	}

	// check if report.ID is on the RFC4122 version 4 format with no urn prefix:
	if strings.HasPrefix(m.ID, "urn:") {
		return false, errors.New("expected no urn: on report ID")
	}

	var u uuid.UUID
	u, err = uuid.FromString(m.ID)

	if err != nil || u.Version() != 4 || u.Variant() != uuid.VariantRFC4122 {
		return false, errors.New("invalid UUID")
	}

	m.ID = strings.ToLower(u.String())

	ts, err := time.Parse(time.RubyDate, m.Timestamp)

	if err != nil {
		return false, errwrap.Wrapf("invalid diagnostics timestamp: {{err}}", err)
	}

	m.TimestampDB = timejson.RubyDate(ts)

	conn := db.Conn()

	stmt, err := conn.PreparexContext(ctx, `
INSERT INTO metrics (
	"id", "type", "text", "tags", "extra", "pid", "sid",
	"timestamp", "version", "os", "arch",
	"request_id", "sync_ip", "sync_location", "timestamp_db")
	VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
	)
	ON CONFLICT DO NOTHING
`)

	if err != nil {
		return false, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	var args = []interface{}{
		m.ID,
		m.Type,
		m.Text,
		m.Tags,
		m.Extra,
		m.PID,
		m.SID,
		m.Timestamp,
		m.Version,
		m.OS,
		m.Arch,
		m.RequestID,
		m.SyncIP,
		m.SyncLocation,
		m.TimestampDB,
	}

	res, err := stmt.ExecContext(ctx, args...)

	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()

	return rows != 0, err
}

// Filter sets the filter settings
type Filter struct {
	Type       string
	Text       string
	Version    string
	NotVersion bool

	Page    int
	PerPage int
}

// Changed tells if values are not default (besides pagination)
func (f Filter) Changed() bool {
	if f.Type != "" || f.Text != "" || f.Version != "" || f.NotVersion {
		return true
	}

	return false
}

// Count reports.
func Count(ctx context.Context, f Filter) (int, error) {
	var q = []string{"SELECT COUNT(id) FROM metrics"}
	var args, where = filter(f)

	if len(where) != 0 {
		q = append(q, "WHERE", where)
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

// List metrics
func List(ctx context.Context, f Filter) (ms []Metric, err error) {
	var q = []string{`SELECT
	id, type, text, tags, extra, pid, sid, timestamp,
	version, os, arch, sync_time, request_id,
	sync_ip, sync_location, timestamp_db FROM metrics`}

	if f.Page == 0 {
		f.Page = 1
	}

	var args, where = filter(f)
	var pos = len(args) + 1

	if len(where) != 0 {
		q = append(q, "WHERE", where)
	}

	var order = fmt.Sprintf("ORDER BY sync_time DESC LIMIT $%d OFFSET $%d", pos, pos+1)
	args = append(args, f.PerPage, (f.Page-1)*f.PerPage)
	pos++

	q = append(q, order)

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
		var m Metric
		err = sqlstruct.Scan(&m, rows)

		if err != nil {
			return nil, err
		}

		ms = append(ms, m)
	}

	return ms, nil
}

func filter(f Filter) (args []interface{}, where string) {
	var pos = len(args) + 1
	var w = []string{}

	if f.Type != "" {
		w = append(w, fmt.Sprintf("type = $%d", pos))
		pos++
		args = append(args, f.Type)
	}

	if f.Text != "" {
		w = append(w, fmt.Sprintf("text ILIKE $%d", pos))
		pos++
		args = append(args, `%`+f.Text+`%`)
	}

	if f.Version != "" || f.NotVersion {
		vop := "="

		if f.NotVersion {
			vop = "!="
		}

		switch f.Version {
		case "":
			w = append(w, "version is NULL")
		default:
			w = append(w, fmt.Sprintf("version %s $%d", vop, pos))
			pos++
			args = append(args, f.Version)

		}
	}

	return args, strings.Join(w, " AND ")
}

// Get metrics entry
func Get(ctx context.Context, id string) (m Metric, err error) {
	var q = `SELECT
	id, type, text, tags, extra, pid, sid, timestamp,
	version, os, arch, sync_time, request_id,
	sync_ip, sync_location, timestamp_db FROM metrics WHERE id = $1`

	conn := db.Conn()
	stmt, err := conn.PreparexContext(ctx, q)

	if err != nil {
		return m, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	row := stmt.QueryRowxContext(ctx, id)

	if err = row.Err(); err != nil {
		return m, err
	}

	err = row.StructScan(&m)
	return m, err
}

// Type structure.
type Type struct {
	Type   string `db:"type"`
	Number int    `db:"number"`
}

// Types of metrics (limited to 500 firsts).
func Types(ctx context.Context) (ts []Type, err error) {
	var q = `SELECT type, COUNT(type) as number FROM metrics GROUP BY type ORDER BY COUNT(type) DESC`

	conn := db.Conn()
	stmt, err := conn.PreparexContext(ctx, q)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryxContext(ctx)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var t Type
		err = sqlstruct.Scan(&t, rows)

		if err != nil {
			return nil, err
		}

		ts = append(ts, t)
	}

	return ts, nil
}

// Versions of the software.
func Versions(ctx context.Context) (versions []string, err error) {
	// BUG(henvic): x.y.z-alpha would appear before x.y.z-beta, etc.
	var q = `SELECT version FROM metrics GROUP BY version
	ORDER BY
	string_to_array(regexp_replace(version, '[^0-9.]', '', 'g'), '.')::int[] DESC,
	version`

	conn := db.Conn()
	stmt, err := conn.PreparexContext(ctx, q)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryxContext(ctx)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var v string
		err = rows.Scan(&v)

		if err != nil {
			return nil, err
		}

		versions = append(versions, v)
	}

	return versions, nil
}

// MissingGeolocation returns a list of IPs where geolocation is missing.
func MissingGeolocation(ctx context.Context) (ips []string, err error) {
	var q = `SELECT sync_ip FROM metrics WHERE sync_location IS NULL GROUP BY sync_ip`

	conn := db.Conn()
	stmt, err := conn.PreparexContext(ctx, q)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	rows, err := stmt.QueryxContext(ctx)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var ip string
		err = rows.Scan(&ip)

		if err != nil {
			return nil, err
		}

		ips = append(ips, ip)
	}

	return ips, nil
}

// AddGeolocationIP adds geolocation info to metrics of a given IP without it.
func AddGeolocationIP(ctx context.Context, ip string) (updated int64, err error) {
	// update geolocation for the given IP, if necessary.
	if _, err = geolocation.Get(ctx, ip); err != nil {
		return 0, err
	}

	var q = `UPDATE metrics
	SET sync_location = (SELECT cache FROM geolocation WHERE ip = $1)
	WHERE sync_ip = $1 AND sync_location IS NULL`

	conn := db.Conn()

	stmt, err := conn.PreparexContext(ctx, q)

	if err != nil {
		return 0, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	res, err := stmt.ExecContext(ctx, ip)

	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}
