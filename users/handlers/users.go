package usershandlers

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	"github.com/henvic/climetrics/server"
	"github.com/henvic/climetrics/us"
	"github.com/henvic/climetrics/users"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/bcrypt"
)

var router = server.Instance.Mux
var usernameRegex = regexp.MustCompile("^[a-z0-9][a-z0-9]*$")

func init() {
	router().Handle("/users", server.AuthenticatedHandler(usersHandler))
	router().Handle("/users/add", server.AuthenticatedHandler(createHandler))
	router().Handle("/users/{user_id}", server.AuthenticatedHandler(editHandler))
}

func usersHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	var query = r.URL.Query()
	var show = "active"

	if len(query.Get("show")) != 0 {
		show = query.Get("show")
	}

	f := users.Filter{}

	switch show {
	case "active":
		f.Active = true
	case "all":
	default:
		server.ErrorHandler(w, r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	users, err := users.List(r.Context(), f)

	if err != nil {
		server.ErrorHandler(w, r, "Can't get users list", http.StatusInternalServerError)
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	var t = &server.Template{
		Title:     "Users",
		Section:   "users",
		Filenames: []string{"gui/users/users.html"},
		Data: map[string]interface{}{
			"Users":     users,
			"Operators": []string{"all", "active"},
			"Show":      show,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func createHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	if r.Method == http.MethodGet {
		var t = &server.Template{
			Title:          "Add an user",
			Section:        "users",
			Filenames:      []string{"gui/users/create.html"},
			Data:           map[string]interface{}{},
			Request:        r,
			ResponseWriter: w,
		}

		t.Respond()
		return
	}

	var username = r.PostFormValue("username")
	var email = r.PostFormValue("email")
	var password = r.PostFormValue("password")

	if !usernameRegex.MatchString(username) {
		server.ErrorHandler(w, r, "Missing or invalid username", http.StatusBadRequest)
		return
	}

	if email == "" {
		server.ErrorHandler(w, r, "Missing email parameter", http.StatusBadRequest)
		return
	}

	if password == "" {
		server.ErrorHandler(w, r, "Missing password parameter", http.StatusBadRequest)
		return
	}

	var role = r.PostFormValue("role")

	switch role {
	case "admin", "member", "revoked":
	default:
		server.ErrorHandler(w, r, "Missing / invalid role parameter", http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	if err != nil {
		server.ErrorHandler(w, r, "Error generating password with bcrypt", http.StatusInternalServerError)
		return
	}

	var user = users.User{
		UserID:   uuid.NewV4().String(),
		Username: username,
		Email:    email,
		Role:     role,
		Password: string(hash),
	}

	if err := users.Create(r.Context(), user); err != nil {
		server.ErrorHandler(w, r, "Internal Server Error: saving user", http.StatusInternalServerError)
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}

func editHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	vars := mux.Vars(r)
	userID, ok := vars["user_id"]

	if !ok {
		server.ErrorHandler(w, r, "Missing user ID parameter", http.StatusBadRequest)
		return
	}

	var user, err = users.Get(r.Context(), userID)

	if err != nil {
		server.ErrorHandler(w, r, fmt.Sprintf("Internal Server Error: %v", err), http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodGet:
		editHandlerGetHandler(w, r, s, user)
	case http.MethodPost:
		editHandlerPostHandler(w, r, s, user)
		return
	default:
		server.ErrorHandler(w, r, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}
}

func editHandlerGetHandler(w http.ResponseWriter, r *http.Request, s us.Session, user users.User) {
	var t = server.Template{
		Title:     "Edit user access",
		Section:   "users",
		Filenames: []string{"gui/users/edit.html"},
		Data: map[string]interface{}{
			"User": user,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func editHandlerPostHandler(w http.ResponseWriter, r *http.Request, s us.Session, user users.User) {
	if err := r.ParseForm(); err != nil {
		server.ErrorHandler(w, r, "Internal Server Error: parsing user edit form", http.StatusInternalServerError)
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
	}

	var username = r.PostFormValue("username")

	if !usernameRegex.MatchString(username) {
		server.ErrorHandler(w, r, "Missing or invalid username", http.StatusBadRequest)
		return
	}

	var email = r.PostFormValue("email")

	if !strings.Contains(email, "@") {
		server.ErrorHandler(w, r, "Internal Server Error: email is invalid", http.StatusInternalServerError)
		return
	}

	var role = r.PostFormValue("role")

	switch role {
	case "admin", "member", "revoked":
	default:
		server.ErrorHandler(w, r,
			fmt.Sprintf("Internal Server Error: role is not recognized: %v", role),
			http.StatusInternalServerError)
		return
	}

	if role == "revoked" && s.User.UserID == user.UserID {
		server.ErrorHandler(w, r,
			fmt.Sprintf("Internal Server Error: role is not recognized: %v", role),
			http.StatusNotAcceptable)
		return
	}

	user.Username = username
	user.Email = email
	user.Role = role

	var password = r.PostFormValue("password")

	if password != "" {
		var hash, err = bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

		if err != nil {
			server.ErrorHandler(w, r, "Error generating password with bcrypt", http.StatusInternalServerError)
			return
		}

		user.Password = string(hash)
	}

	if err := users.Update(r.Context(), user); err != nil {
		server.ErrorHandler(w, r, "Internal Server Error: saving user", http.StatusInternalServerError)
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
		return
	}

	http.Redirect(w, r, "/users", http.StatusSeeOther)
}
