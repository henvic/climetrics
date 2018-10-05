package diagnosticshandlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/henvic/climetrics/diagnostics"
	"github.com/henvic/climetrics/server"
	"github.com/henvic/climetrics/us"
	log "github.com/sirupsen/logrus"
)

var router = server.Instance.Mux

func init() {
	router().HandleFunc("/diagnostics/report", reportHandler)
	server.Protected.Unsafe("/diagnostics/report")

	router().Handle("/diagnostics",
		server.AuthenticatedHandler(listOrReadHandler))

	router().Handle("/diagnostics/{id}",
		server.AuthenticatedHandler(readHandler))
}

func listOrReadHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	if ids, ok := r.URL.Query()["id"]; ok && len(ids) != 0 {
		id := ids[0]
		http.Redirect(w, r, fmt.Sprintf("/diagnostics/%s", id), http.StatusSeeOther)
		return
	}

	listHandler(w, r, s)
}

func listHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	var query = r.URL.Query()
	var username = ""

	if usernames, ok := query["username"]; ok && len(usernames) != 0 {
		username = usernames[0]
	}

	var page = 1
	var err error

	if len(query["page"]) != 0 {
		page, err = strconv.Atoi(query["page"][0])

		if err != nil {
			server.ErrorHandler(w, r, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if page == 0 {
			page = 1
		}
	}

	var usernameOp = diagnostics.Contains

	if query.Get("op") != "" {
		usernameOp = diagnostics.Op(query.Get("op"))
	}

	f := diagnostics.Filter{
		Username:         username,
		UsernameOperator: diagnostics.Op(usernameOp),

		Page:    page,
		PerPage: 50,
	}

	count, err := diagnostics.Count(r.Context(), f)

	if err != nil {
		log.Errorf("failed to count number of diagnostics: %+v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	list, err := diagnostics.List(r.Context(), f)

	if err != nil {
		log.Errorf("failed to list diagnostics: %+v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var maxPage = count / f.PerPage

	if count%f.PerPage != 0 {
		maxPage++
	}

	var operators = []string{
		string(diagnostics.Contains),
		string(diagnostics.Equal),
		string(diagnostics.Like),
	}

	var t = &server.Template{
		Title:     "Diagnostics",
		Section:   "diagnostics",
		Filenames: []string{"gui/diagnostics/list.html"},
		Data: map[string]interface{}{
			"List":      list,
			"Count":     count,
			"MaxPage":   maxPage,
			"Operators": operators,
			"Filter":    f,
			"URL":       r.URL,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func readHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	vars := mux.Vars(r)
	var report, err = diagnostics.Get(r.Context(), vars["id"])

	if err == sql.ErrNoRows {
		server.ErrorHandler(w, r, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	if err != nil {
		log.Error(err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var t = &server.Template{
		Title:     "Diagnostics " + report.ID,
		Section:   "diagnostics",
		Filenames: []string{"gui/diagnostics/report.html"},
		Data: map[string]interface{}{
			"Report":      report,
			"Diagnostics": terminal2html(report.Report),
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func terminal2html(s string) template.HTML {
	s = strings.Replace(html.EscapeString(s), "\n", "<br />", -1)
	return template.HTML(s)
}

func reportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		server.ErrorHandler(w, r,
			http.StatusText(http.StatusMethodNotAllowed),
			http.StatusMethodNotAllowed)
		return
	}

	if !strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		server.ErrorHandler(w, r,
			http.StatusText(http.StatusUnsupportedMediaType),
			http.StatusUnsupportedMediaType)
		return
	}

	var report diagnostics.Report
	var err = json.NewDecoder(r.Body).Decode(&report)

	if err != nil {
		server.ErrorHandler(w, r,
			http.StatusText(http.StatusUnsupportedMediaType),
			http.StatusUnsupportedMediaType)
		return
	}

	err = diagnostics.Create(r.Context(), report)

	if err != nil {
		server.ErrorHandler(w, r,
			err.Error(),
			http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", fmt.Sprintf("/diagnostics/%s", report.ID))
	w.WriteHeader(http.StatusCreated)
}
