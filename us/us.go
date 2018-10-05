package us

import (
	"github.com/gorilla/sessions"
	"github.com/henvic/climetrics/users"
)

// Session for the requests.
type Session struct {
	Session *sessions.Session
	User    users.User
}
