package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/hashicorp/errwrap"
	"github.com/henvic/climetrics/db"
	"github.com/kisielk/sqlstruct"
	log "github.com/sirupsen/logrus"
)

// UserSessionName is used by the cookie
const UserSessionName = "sclimetrics"

// SessionStore for the cookie store
var SessionStore *sessions.CookieStore // @todo FileSystem?

var router = mux.NewRouter()

// Params of the service
type Params struct {
	Address            string
	DSN                string
	UserSessionPrefix  string
	SessionStoreSecret string

	ExposeDebug bool
}

// ProtectedHandler does CSRF protection.
type ProtectedHandler struct {
	secret []byte

	unsafe map[string]bool
	m      sync.RWMutex
}

// Unsafe marks a endpoint as unprotected by CSRF.
func (p *ProtectedHandler) Unsafe(path string) {
	p.m.Lock()
	defer p.m.Unlock()
	p.unsafe[path] = true
}

func (p *ProtectedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p.m.RLock()
	var unsafe = p.unsafe != nil && p.unsafe[r.URL.Path]
	p.m.RUnlock()

	if unsafe {
		r = csrf.UnsafeSkipCheck(r)
	}

	var pr = csrf.Protect(p.secret,
		csrf.Secure(false),
		csrf.ErrorHandler(csrfErrorHandler{}),
	)(router)

	pr.ServeHTTP(w, r)
}

// Protected handler.
var Protected = &ProtectedHandler{
	secret: secret(),

	unsafe: map[string]bool{},
}

// Instance of the server
var Instance = &Server{
	httpServer: &http.Server{
		Handler: Protected,
	},
	mux: router,
}

func init() {
	sqlstruct.NameMapper = sqlstruct.ToSnakeCase
	router.StrictSlash(true)
}

func secret() []byte {
	// TODO(henvic): make it stick during restarts
	b := make([]byte, 32)

	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return b
}

// Start server for climetrics
func Start(ctx context.Context, params Params) error {
	return Instance.Serve(ctx, params)
}

// Server for handling requests
type Server struct {
	ctx context.Context

	params Params

	mux *mux.Router

	httpServer *http.Server
}

// Mux of the server
func (s *Server) Mux() *mux.Router {
	return s.mux
}

// Params of the server
func (s *Server) Params() Params {
	return s.params
}

// Serve handlers
func (s *Server) Serve(ctx context.Context, params Params) error {
	s.ctx = ctx
	s.params = params

	if err := db.Load(ctx, params.DSN); err != nil {
		return err
	}

	SessionStore = sessions.NewCookieStore(
		[]byte(params.SessionStoreSecret),
	)

	return s.http()
}

func getAddr(a string) string {
	l := strings.LastIndex(a, ":")

	if l == -1 && len(a) <= l {
		return a
	}

	return "http://localhost:" + a[l+1:]
}

// Serve HTTP requests
func (s *Server) http() error {
	listener, err := net.Listen("tcp", s.params.Address)
	log.Infof("Starting server on %v", getAddr(listener.Addr().String()))

	if err != nil {
		return err
	}

	ec := make(chan error, 1)

	go func() {
		<-s.ctx.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil && err != context.Canceled {
			ec <- errwrap.Wrapf("can't shutdown server properly: {{err}}", err)
		}
	}()

	go func() {
		ec <- s.httpServer.Serve(listener)
	}()

	e := <-ec

	if e == http.ErrServerClosed {
		fmt.Println()
		log.Info("Server shutting down gracefully.")
		return nil
	}

	return e
}
