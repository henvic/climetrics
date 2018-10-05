package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	humanize "github.com/dustin/go-humanize"
	"github.com/henvic/climetrics/timejson"
	"github.com/henvic/climetrics/us"

	"github.com/gorilla/csrf"
	"github.com/gorilla/securecookie"
)

// Template to use
type Template struct {
	Title          string
	Section        string
	Filenames      []string
	Data           interface{}
	Request        *http.Request
	ResponseWriter http.ResponseWriter
}

const base = "gui/template.html"

var basicFunctions = template.FuncMap{
	"json": func(v interface{}) string {
		a, _ := json.Marshal(v)
		return string(a)
	},
	"split": strings.Split,
	"join":  strings.Join,
	"title": strings.Title,
	"lower": strings.ToLower,
	"upper": strings.ToUpper,

	"humanizeTime": humanizeTime,
	"paginator":    paginator,
	"add":          add,
}

func (t *Template) isSectionActiveFunc(v interface{}) bool {
	return v.(string) == t.Section
}

func (t *Template) printSectionActiveFunc(section string) string {
	return t.printValueIfSectionIsActiveFunc(section, " active ")
}

func (t *Template) printValueIfSectionIsActiveFunc(section string, value string) string {
	if t.isSectionActiveFunc(section) {
		return " active "
	}

	return ""
}

func humanizeTime(t interface{}) (string, error) {
	if tv, ok := t.(time.Time); ok {
		return humanize.Time(tv), nil
	}

	if tv, ok := t.(timejson.RubyDate); ok {
		return humanize.Time(time.Time(tv)), nil
	}

	return "", errors.New("invalid time format")
}

func add(a, b int) int {
	return a + b
}

func paginator(u url.URL, page int) template.HTML {
	q := u.Query()
	q.Del("page")
	u.RawQuery = q.Encode()

	if page <= 1 {
		return template.HTML(u.String())
	}

	switch u.RawQuery {
	case "":
		u.RawQuery = fmt.Sprintf("page=%d", page)
	default:
		u.RawQuery += fmt.Sprintf("&page=%d", page)
	}

	return template.HTML(u.String())
}

// SessionCtx is a pointer to the session data.
type SessionCtx struct{}

// Execute template
func (t *Template) Execute() error {
	var files = []string{base}
	files = append(files, t.Filenames...)

	req := t.Request

	var to = template.New("").Funcs(basicFunctions).Funcs(template.FuncMap{
		"isSectionActive":           t.isSectionActiveFunc,
		"printSectionActive":        t.printSectionActiveFunc,
		"printValueIfSectionActive": t.printValueIfSectionIsActiveFunc,
	})

	_, err := to.ParseFiles(files...)

	if err != nil {
		return err
	}

	reqCtx := req.Context()
	si := reqCtx.Value(SessionCtx{})

	var s us.Session

	if sp, ok := si.(us.Session); ok {
		s = sp
	}

	var values = map[string]interface{}{
		"Title": t.Title,
		"Data":  t.Data,

		"Session": s.Session,
		"User":    s.User,

		csrf.TemplateTag: csrf.TemplateField(req),
	}

	return to.ExecuteTemplate(t.ResponseWriter, "base", values)
}

// Respond request
func (t *Template) Respond() {
	err := t.Execute()

	switch err.(type) {
	case securecookie.MultiError:
		ExpireCookie(t.ResponseWriter)
		http.Error(t.ResponseWriter, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(t.ResponseWriter, "Internal Server Error: template parsing", http.StatusInternalServerError)
		_, _ = fmt.Fprintf(os.Stderr, "%+v\n", err)
	}
}
