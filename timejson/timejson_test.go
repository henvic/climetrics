package timejson

import (
	"encoding/json"
	"testing"
	"time"
)

func TestRubyDateJSONMarshal(t *testing.T) {
	tu := time.Unix(1000000000, 0)

	loc, err := time.LoadLocation("Asia/Shanghai")

	if err != nil {
		panic(err)
	}

	tu = tu.In(loc)

	var past = RubyDate(tu)

	var data []byte
	data, err = json.Marshal(past)

	var want = `"Sun Sep 09 09:46:40 +0800 2001"`

	if err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	if string(data) != want {
		t.Errorf("Expected time to be %v, got %v instead", want, string(data))
	}
}

func TestRubyDateJSONUnmarshal(t *testing.T) {
	var data = `"Thu Sep 27 00:32:23 +0200 2018"`
	var want = int64(1538001143)
	var r RubyDate

	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	var rt = time.Time(r)
	var got = rt.Unix()

	if got != want {
		t.Errorf("Expected Unix time %v, got %v instead", want, got)
	}
}

func TestRubyDateJSONUnmarshalNull(t *testing.T) {
	var data = `null`
	var r RubyDate

	if err := json.Unmarshal([]byte(data), &r); err != nil {
		t.Errorf("Expected no error, got %v instead", err)
	}

	var rt = time.Time(r)

	if (rt != time.Time{}) {
		t.Error("Expected null to be ignored")
	}
}
