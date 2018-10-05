package authhandlers

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/schema"
	"github.com/henvic/climetrics/auth"
	"github.com/henvic/climetrics/server"
	"github.com/henvic/climetrics/us"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var router = server.Instance.Mux

type loginForm struct {
	Username string
	Password string
}

func init() {
	router().HandleFunc("/login", loginHandler)
	router().Handle("/logout", server.AuthenticatedHandler(logoutHandler))
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	var session, err = server.SessionStore.Get(r, server.UserSessionName)

	if err != nil {
		log.Errorf("Session store error: %v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if r.Method == http.MethodGet {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method != http.MethodPost {
		server.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if ok, iok := session.Values["authenticated"].(bool); iok && ok {
		server.ErrorHandler(w, r, "Conflict: user is already signed in", http.StatusConflict)
		return
	}

	if err = r.ParseForm(); err != nil {
		server.ErrorHandler(w, r, "Invalid form", http.StatusBadRequest)
		return
	}

	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	login := loginForm{}

	if err = decoder.Decode(&login, r.PostForm); err != nil {
		server.ErrorHandler(w, r, fmt.Sprintf("Error decoding request body: %v", err), http.StatusBadRequest)
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	if len(login.Username) == 0 {
		server.ErrorHandler(w, r, "Wrong credentials.", http.StatusUnauthorized)
		return
	}

	var a auth.Authentication
	a, err = auth.Get(r.Context(), login.Username)

	if err == sql.ErrNoRows {
		server.ErrorHandler(w, r, "Wrong credentials.", http.StatusUnauthorized)
		return
	}

	if err != nil {
		log.Errorf("Error getting authentication data from DB: %v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if err = bcrypt.CompareHashAndPassword([]byte(a.Password), []byte(login.Password)); err != nil {
		server.ErrorHandler(w, r, "Wrong credentials.", http.StatusUnauthorized)
		return
	}

	session.Values["user_id"] = a.UserID
	session.Values["authenticated"] = true

	if err = session.Save(r, w); err != nil {
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func logoutHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	if r.Method != http.MethodPost {
		server.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	session := s.Session
	session.Values = map[interface{}]interface{}{}
	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   server.UserSessionName,
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
