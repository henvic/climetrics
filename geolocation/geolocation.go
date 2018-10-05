package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/henvic/climetrics/db"
	"github.com/jmoiron/sqlx"
)

// Host for the service.
const Host = "https://ipinfo.io/"

// TTL is how long a cache entry is considered fresh (stored in PostgreSQL style)
const ttl = "48 hours"

const backoff429 = 3 * time.Hour // backoff for 'Too Many Requests'

type cache struct {
	IP        string    `db:"ip"`
	Cache     []byte    `db:"cache"`
	Timestamp time.Time `db:"timestamp"`
}

// Get geolocation for a given IP. Updates cache if needed.
func Get(ctx context.Context, ip string) (data []byte, err error) {
	data, err = Cached(ctx, ip)

	if err != nil {
		return Refresh(ctx, ip)
	}

	return data, nil
}

// Cached returns info about an IP that has been cached.
// If data is not fresh, it updates info before returning.
func Cached(ctx context.Context, ip string) (data []byte, err error) {
	var q = `SELECT ip, cache, timestamp FROM geolocation WHERE ip = $1 AND
	timestamp >= NOW() - $2::interval`

	conn := db.Conn()

	var stmt *sqlx.Stmt
	stmt, err = conn.PreparexContext(ctx, q)

	if err != nil {
		return nil, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	row := stmt.QueryRowxContext(ctx, ip, ttl)

	if err = row.Err(); err != nil {
		return nil, err
	}

	var c cache
	err = row.StructScan(&c)
	return c.Cache, err
}

// Refresh cache.
func Refresh(ctx context.Context, ip string) ([]byte, error) {
	b, err := fetch(ctx, ip)

	if err != nil {
		return nil, err
	}

	_, err = upsert(ctx, ip, json.RawMessage(b))
	return b, err
}

func upsert(ctx context.Context, ip string, d json.RawMessage) (created bool, err error) {
	conn := db.Conn()

	stmt, err := conn.PreparexContext(ctx, `INSERT INTO geolocation
	(ip, cache, timestamp)
	VALUES ($1, $2, CURRENT_TIMESTAMP)
	ON CONFLICT ON CONSTRAINT geolocation_pkey
	DO UPDATE SET cache = $2, timestamp = CURRENT_TIMESTAMP
`)

	if err != nil {
		return false, err
	}

	defer func() {
		_ = stmt.Close()
	}()

	var args = []interface{}{
		ip,
		d,
	}

	res, err := stmt.ExecContext(ctx, args...)

	if err != nil {
		return false, err
	}

	rows, err := res.RowsAffected()

	return rows != 0, err
}

func fetch(ctx context.Context, ip string) (b []byte, err error) {
	if err = green429(); err != nil {
		return nil, err
	}

	u, err := url.Parse(Host)

	if err != nil {
		return nil, err
	}

	u.Path = fmt.Sprintf("/%s/json", url.PathEscape(ip))

	var c = http.DefaultClient

	var req *http.Request
	req, err = http.NewRequest("GET", u.String(), nil)

	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "climetrics/alpha (+https://github.com/henvic/climetrics)")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	req = req.WithContext(ctx)

	var resp *http.Response
	resp, err = c.Do(req)

	if err != nil {
		return nil, err
	}

	defer func() {
		ec := resp.Body.Close()

		if err == nil {
			err = ec
		}
	}()

	if resp.StatusCode == http.StatusTooManyRequests {
		slowdown429() // could as well check the Retry-After header

		if err = green429(); err != nil {
			return nil, err
		}
	}

	if resp.StatusCode < 200 && resp.StatusCode > 299 {
		return nil, fmt.Errorf("response has status code %d", resp.StatusCode)
	}

	b, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		return nil, err
	}

	// validate JSON as a last step.
	var v map[string]interface{}

	err = json.Unmarshal(b, &v)

	if err != nil {
		return nil, err
	}

	v["cached"] = time.Now().Format(time.RFC3339)
	return json.Marshal(v)
}

var deadline429 = time.Now()
var m429 sync.RWMutex

func green429() error {
	// TODO(henvic): Add a counter watch to fire warnings (or soft errors) if reaching near the limit.
	m429.RLock()
	defer m429.RUnlock()

	if time.Since(deadline429) <= 0 {
		return fmt.Errorf("too many requests: canceling any requests until %v", deadline429)
	}

	return nil
}

func slowdown429() {
	m429.Lock()
	defer m429.Unlock()
	deadline429 = time.Now().Add(backoff429)
}
