package server

import (
	"net/http"
)

// ExpireCookie sets the cookie to expire.
func ExpireCookie(w http.ResponseWriter) {
	c := http.Cookie{
		Name:   UserSessionName,
		MaxAge: -1,
	}

	http.SetCookie(w, &c)
}
