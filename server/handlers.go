package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/hashicorp/errwrap"
	"github.com/henvic/climetrics/us"
	"github.com/henvic/climetrics/users"
	log "github.com/sirupsen/logrus"
)

func init() {
	var mux = Instance.Mux()
	mux.Handle("/", PublicHandler(homeHandler))
	mux.PathPrefix("/static").HandlerFunc(staticHandler)
	mux.NotFoundHandler = &notFoundHandler{}
}

type csrfErrorHandler struct{}

func (csrfErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err = csrf.FailureReason(r)
	var es = "unknown error"

	if err != nil {
		es = err.Error()
	}

	ErrorHandler(w, r, es, http.StatusForbidden)
}

// AuthenticatedHandler is a handler for an authenticated-only request
type AuthenticatedHandler func(w http.ResponseWriter, r *http.Request, s us.Session)

// PublicHandler is a handler for public requests on the browser
type PublicHandler func(w http.ResponseWriter, r *http.Request, s us.Session)

func (h AuthenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var s us.Session
	w, r, s, err = serveHTTP(w, r)

	if err != nil {
		log.Debug(err)
		bareErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	session := s.Session

	if _, ok := session.Values["authenticated"]; !ok {
		ErrorHandler(w, r, "Access restricted. Please log in.", http.StatusUnauthorized)
		return
	}

	h(w, r, s)
}

func (h PublicHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var err error
	var s us.Session
	w, r, s, err = serveHTTP(w, r)

	if err != nil {
		log.Debug(err)
		bareErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	h(w, r, s)
}

func serveHTTP(w http.ResponseWriter, r *http.Request) (http.ResponseWriter, *http.Request, us.Session, error) {
	var session, err = SessionStore.Get(r, UserSessionName)
	s := us.Session{}

	if err != nil {
		deleteSessionCookie(w)
		return w, r, s, errwrap.Wrapf("session error: {{err}}", err)
	}

	s.Session = session

	if _, ok := session.Values["authenticated"]; !ok {
		return w, r, s, nil
	}

	userID, ok := session.Values["user_id"].(string)

	if !ok {
		deleteSessionCookie(w)
		return w, r, s, errors.New("user_id is not a string")
	}

	var u users.User
	u, err = users.Get(r.Context(), userID)

	if err != nil {
		deleteSessionCookie(w)
		return w, r, s, errwrap.Wrapf("can't retrieve user info: {{err}}", err)
	}

	s.User = u
	ctx := context.WithValue(r.Context(), SessionCtx{}, s)
	return w, r.WithContext(ctx), s, nil
}

func deleteSessionCookie(w http.ResponseWriter) {
	w.Header().Set("Set-Cookie", fmt.Sprintf(
		"%s=; path=/; expires=Thu, 01 Jan 1970 00:00:00 GMT", UserSessionName))
}

func homeHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	var t = &Template{
		Title:          "CLI metrics",
		Section:        "home",
		Filenames:      []string{"gui/home/home.html"},
		Data:           map[string]interface{}{},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func staticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, fmt.Sprintf("gui/%v", r.URL.Path))
}

type notFoundHandler struct{}

func (n *notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ErrorHandler(w, r, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

// ErrorHandler for errors
func ErrorHandler(w http.ResponseWriter, r *http.Request, e string, code int) {
	var err error
	w, r, _, err = serveHTTP(w, r)

	if err != nil {
		log.Debug(err)
		bareErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	a := r.Header.Get("Accept")
	ua := r.Header.Get("User-Agent")

	if strings.Contains(a, "application/json") || (a == "*/*" && strings.HasPrefix(ua, "curl/")) {
		jsonErrorHandler(w, r, e, code)
		return
	}

	var t = Template{
		Title:     fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Filenames: []string{"gui/errors/error.html"},
		Data: map[string]interface{}{
			"StatusText": http.StatusText(code),
			"Message":    e,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func jsonErrorHandler(w http.ResponseWriter, r *http.Request, e string, code int) {
	var m = map[string]interface{}{
		"status":  code,
		"message": e,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf8")
	w.WriteHeader(code)

	mj, _ := json.MarshalIndent(&m, "", "    ")

	_, _ = fmt.Fprintf(w, "%s\n", mj)
}

func bareErrorHandler(w http.ResponseWriter, r *http.Request, e string, code int) {
	w.WriteHeader(code)
	_, _ = fmt.Fprintf(w, "%s\n", e)
}
