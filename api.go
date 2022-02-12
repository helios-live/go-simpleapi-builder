package apicontroller

import (
	"context"
	"log"
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
	router               *mux.Router
	server               *http.Server
	useDefaultMiddleware bool
	AuthCallback         AuthCallback
}
type key int

const (
	// KeyAuthID returns the context Value of the Auth set by the defaultAuthMiddleware func on successful auth
	KeyAuthID key = iota
	// ...
)

// NewController creates a new HTTP API controller
func NewController() *Controller {
	c := Controller{
		router:               mux.NewRouter(),
		useDefaultMiddleware: true,
	}
	return &c
}

// AddHandler adds a handler
func (c *Controller) AddHandler(path string, fn HTTPHandler, methods ...string) {
	c.router.HandleFunc(path, fn).Methods(methods...)
}

// Run runs the controller and the listener
func (c *Controller) Run(addr string) {

	allowed_origins := os.Getenv("ORIGIN_ALLOWED")
	if len(allowed_origins) == 0 {
		allowed_origins = "*"
	}
	headersOk := handlers.AllowedHeaders([]string{"X-Requested-With", "Authorization"})
	originsOk := handlers.AllowedOrigins([]string{allowed_origins})
	methodsOk := handlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	if c.useDefaultMiddleware {
		c.router.Use(c.defaultAuthMiddleware)
	}
	c.server = &http.Server{Addr: addr, Handler: handlers.CORS(headersOk, originsOk, methodsOk)(c.router)}

	log.Println("Running API at http://" + addr)
	// log.Fatal(http.ListenAndServe(addr, nil))
	if err := c.server.ListenAndServe(); err != nil {
		// handle err
	}
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
		if parts[0] != "Bearer" {
			w.Header().Add("X-Error", "Only Authorization: Bearer Allowed")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		token := parts[1]

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
