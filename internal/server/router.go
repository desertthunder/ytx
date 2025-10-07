package server

import (
	"net/http"
	"strings"
)

// BasicRouter is a simple HTTP router implementing the [Router] interface.
//
// Uses [http.ServeMux] internally for routing.
type BasicRouter struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

// NewBasicRouter creates a new [BasicRouter] instance.
func NewBasicRouter() *BasicRouter {
	return &BasicRouter{
		mux:         http.NewServeMux(),
		middlewares: []Middleware{},
	}
}

// Use adds [Middleware] to the [Router] instance's middleware stack, applied in the order it's added.
func (r *BasicRouter) Use(middleware ...Middleware) {
	r.middlewares = append(r.middlewares, middleware...)
}

// Handle registers a [Handler] for the specified HTTP method and path.
//
// The handler is wrapped with all registered middleware.
func (r *BasicRouter) Handle(method, path string, handler http.Handler) {
	wrapped := r.Apply(handler)

	methodHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if !strings.EqualFold(req.Method, method) {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		wrapped.ServeHTTP(w, req)
	})

	r.mux.Handle(path, methodHandler)
}

// Handler registers a custom Handler implementation.
//
// All routes returned by [Handler.Routes] are registered with this handler.
func (r *BasicRouter) Handler(handler Handler) {
	wrapped := r.Apply(handler)

	for _, route := range handler.Routes() {
		r.mux.Handle(route, wrapped)
	}
}

// ServeHTTP implements [http.Handler] for the entire router.
func (r *BasicRouter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Apply wraps a handler with all registered middleware.
//
// Middleware is applied in reverse order (last added wraps first).
func (r *BasicRouter) Apply(handler http.Handler) http.Handler {
	wrapped := handler

	for i := len(r.middlewares) - 1; i >= 0; i-- {
		wrapped = r.middlewares[i](wrapped)
	}

	return wrapped
}
