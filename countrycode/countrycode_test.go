package countrycode

import "testing"

func TestGet(t *testing.T) {
	var want = "Iceland"
	var got = Get("IS")

	if got != want {
		t.Errorf("Expected to get %v, got %v instead", want, got)
	}
}

func TestGetFailure(t *testing.T) {
	var want = ""
	var got = Get("Gondor")

	if got != want {
		t.Errorf("Expected to get %v, got %v instead", want, got)
	}
}
