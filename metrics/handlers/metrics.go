package metricshandlers

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/henvic/climetrics/metrics"
	"github.com/henvic/climetrics/server"
	"github.com/henvic/climetrics/us"
	"github.com/lib/pq"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
	"github.com/tomasen/realip"
)

var router = server.Instance.Mux

func init() {
	router().Handle("/metrics", server.AuthenticatedHandler(listHandler))
	router().HandleFunc("/metrics/bulk", bulkAddHandler)
	server.Protected.Unsafe("/metrics/bulk")
	router().Handle("/metrics/{id}", server.AuthenticatedHandler(readHandler))
}

type bulkStats struct {
	RequestID string `json:"request_id"`

	Added int `json:"added"`
	Noop  int `json:"noop"`
	Error int `json:"error"`

	Broken []int `json:"broken_lines,omitempty"`
}

func bulkAddHandler(w http.ResponseWriter, r *http.Request) {
	var requestID = uuid.NewV4().String()

	var b = bulkStats{
		RequestID: requestID,
	}

	s := bufio.NewScanner(r.Body)
	line := 0

	ip := realip.FromRequest(r)

	for s.Scan() {
		line++
		mt := s.Text()
		m, err := unmarshalMetric(mt)
		m.RequestID = requestID
		m.SyncIP = ip

		if err != nil {
			log.Debugf("can't unmarshal metric: %+v: %+v", mt, err)
			b.Error++
			b.Broken = append(b.Broken, line)
			continue
		}

		added, err := metrics.Create(r.Context(), m)

		if err != nil {
			b.Error++
			b.Broken = append(b.Broken, line)

			switch err.(type) {
			case *pq.Error:
				log.Error(err)
			default:
				log.Debugf("can't create metric on DB: %+v: %+v", mt, err)
			}

			continue
		}

		switch added {
		case true:
			b.Added++
		default:
			b.Noop++
		}
	}

	go addGeolocation(ip)

	if err := s.Err(); err != nil {
		log.Error(s)
		server.ErrorHandler(w, r,
			http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", "application/json; charset=utf8")

	bj, _ := json.MarshalIndent(&b, "", "    ")
	_, _ = fmt.Fprintf(w, "%s\n", bj)
}

func addGeolocation(ip string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Debugf("trying to add geolocation for IP %s", ip)

	if _, err := metrics.AddGeolocationIP(ctx, ip); err != nil {
		log.Errorf("can't add geolocation for IP %s: %+v", ip, err)
	}
}

func unmarshalMetric(s string) (m metrics.Metric, err error) {
	err = json.Unmarshal([]byte(s), &m)
	return m, err
}

func listHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	var query = r.URL.Query()

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

	var fType, text, version string

	if len(query["type"]) != 0 {
		fType = query["type"][0]
	}

	if len(query["text"]) != 0 {
		text = query["text"][0]
	}

	if len(query["version"]) != 0 {
		version = query["version"][0]
	}

	f := metrics.Filter{
		Type:       fType,
		Text:       text,
		Version:    version,
		NotVersion: len(query["not-version"]) != 0,

		Page:    page,
		PerPage: 100,
	}

	count, err := metrics.Count(r.Context(), f)

	if err != nil {
		log.Errorf("failed to count number of metrics: %+v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	list, err := metrics.List(r.Context(), f)

	if err != nil {
		log.Errorf("failed to list metrics: %+v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var maxPage = count / f.PerPage

	if count%f.PerPage != 0 {
		maxPage++
	}

	types, err := metrics.Types(r.Context())

	if err != nil {
		log.Errorf("failed to list metrics types: %+v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	versions, err := metrics.Versions(r.Context())

	if err != nil {
		log.Errorf("failed to list metrics versions: %+v", err)
		server.ErrorHandler(w, r, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	var t = &server.Template{
		Title:     "Metrics",
		Section:   "metrics",
		Filenames: []string{"gui/metrics/list.html"},
		Data: map[string]interface{}{
			"List":     list,
			"Count":    count,
			"MaxPage":  maxPage,
			"Filter":   f,
			"URL":      r.URL,
			"Types":    types,
			"Versions": versions,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}

func readHandler(w http.ResponseWriter, r *http.Request, s us.Session) {
	vars := mux.Vars(r)
	var m, err = metrics.Get(r.Context(), vars["id"])

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
		Title:     "Metric entry " + m.ID,
		Section:   "metrics",
		Filenames: []string{"gui/metrics/entry.html"},
		Data: map[string]interface{}{
			"Types": []string{"all", "cmd"},
			"Entry": m,
		},
		Request:        r,
		ResponseWriter: w,
	}

	t.Respond()
}
