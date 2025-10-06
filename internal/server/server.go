// package server contains middleware & handlers for the song migration web service
package server

import (
	"net/http"
)

// Middleware wraps an http.Handler and returns a new http.Handler with additional behavior.
// Common middleware includes logging, authentication, CORS, rate limiting, etc.
type Middleware func(http.Handler) http.Handler

// Handler defines the interface for HTTP request handlers in the song migration service.
// Implementations handle specific endpoints (auth, playlist operations, migrations).
type Handler interface {
	http.Handler      // ServeHTTP handles the HTTP request and writes the response
	Routes() []string // Routes returns the path patterns this handler serves
}

// Router defines the interface for HTTP routing and middleware management.
// Implementations register handlers, apply middleware, and configure the HTTP server.
type Router interface {
	Use(middleware ...Middleware)                     // Use adds middleware to the router's middleware stack
	Handle(method, path string, handler http.Handler) // Handle registers a handler for the specified method and path
	Handler(handler Handler)                          // Handler registers a custom Handler implementation
	ServeHTTP(w http.ResponseWriter, r *http.Request) // ServeHTTP implements http.Handler for the entire router
}
