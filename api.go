package api // import go.ideatocode.tech/api

import (
	"context"
	"encoding/base64"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// HTTPHandler is the handler that is called when path is accessed
type HTTPHandler func(w http.ResponseWriter, r *http.Request)

// AuthCallback is the function called when doing bearer authentication
type AuthCallback func(token string, req *http.Request) (payload interface{}, err error)

// Controller runs the controller
type Controller struct {
	router                 *mux.Router
	server                 *http.Server
	useDefaultMiddleware   bool
	AuthCallback           AuthCallback
	listener               *net.Listener
	allowAnonymousRequests bool
}
type key int

const (
	// KeyAuthID returns the context Value of the Auth set by the defaultAuthMiddleware func on successful auth
	KeyAuthID key = iota
	// ...
)

type SetOptFunc func(*Controller)

// NewController creates a new HTTP API controller
func NewController(list ...SetOptFunc) *Controller {
	c := Controller{
		router:                 mux.NewRouter(),
		useDefaultMiddleware:   true,
		allowAnonymousRequests: false,
	}
	// loop through the list
	for _, fn := range list {
		fn(&c)
	}
	return &c
}

// WithAnonymousRequests allows requests with no auth headers through to the auth middleware
func WithAnonymousRequests() SetOptFunc {
	return func(c *Controller) {
		c.allowAnonymousRequests = true
	}
}

// AddHandler adds a handler
func (c *Controller) AddHandler(path string, fn HTTPHandler, methods ...string) {
	c.router.HandleFunc(path, fn).Methods(methods...)
}

// Run runs the controller and the listener
func (c *Controller) Run(addr string) error {

	allowedOrigins := os.Getenv("ORIGIN_ALLOWED")
	if len(allowedOrigins) == 0 {
		allowedOrigins = "*"
	}
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{allowedOrigins})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	if c.useDefaultMiddleware {
		c.router.Use(c.defaultAuthMiddleware)
	}
	c.server = &http.Server{Addr: addr, Handler: handlers.CORS(headersOk, originsOk, methodsOk)(c.router)}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	c.listener = &ln

	err = c.server.Serve(*c.listener)
	return err

}

// Addr returns the underlying net.Addr that this server listens on
func (c *Controller) Addr() net.Addr {
	return (*c.listener).Addr()
}

// Stop stops the http listener
func (c *Controller) Stop() {
	if c.server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c.server.Shutdown(ctx)
	c.server = nil
}

func (c *Controller) setMiddleware(h ...mux.MiddlewareFunc) {
	c.useDefaultMiddleware = false
	c.router.Use(h...)
}
func (c *Controller) defaultAuthMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// if there's no auth callback then skip auth
		if c.AuthCallback == nil {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		header := r.Header.Get("Authorization")
		parts := strings.Split(header, " ")

		var token string

		switch parts[0] {
		case "Bearer":
			token = parts[1]

		case "Basic":
			data, err := base64.StdEncoding.DecodeString(parts[1])
			if err != nil {
				w.Header().Add("X-Error", "Improper authorization header, failed to decode base64 string")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			token = string(data)
		default:
			if !c.allowAnonymousRequests {
				w.Header().Add("X-Error", "Only Authorization: Bearer Allowed")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}

		fn := c.AuthCallback
		id, err := fn(token, r)
		if err != nil {
			w.Header().Set("X-Error", err.Error())
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx = context.WithValue(ctx, KeyAuthID, id)
		// w.Header().Set("X-ID", id)

		// continue from here
		next.ServeHTTP(w, r.WithContext(ctx))
		return
	})
}
