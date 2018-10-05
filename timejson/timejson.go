package timejson

import (
	"database/sql/driver"
	"time"
)

// RubyDate format.
type RubyDate time.Time

// MarshalJSON is used for writing a JSON value.
func (r RubyDate) MarshalJSON() ([]byte, error) {
	t := time.Time(r)
	return []byte(t.Format(`"` + time.RubyDate + `"`)), nil
}

// UnmarshalJSON is used for parsing a JSON value.
func (r *RubyDate) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if string(data) == "null" {
		return nil
	}

	var rt, err = time.Parse(`"`+time.RubyDate+`"`, string(data))
	*r = RubyDate(rt)
	return err
}

// Scan implements the Scanner interface.
func (r *RubyDate) Scan(value interface{}) error {
	*r = RubyDate(value.(time.Time))
	return nil
}

// Value implements the driver Valuer interface.
func (r RubyDate) Value() (driver.Value, error) {
	return time.Time(r), nil
}
