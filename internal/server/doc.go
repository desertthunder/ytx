// Package server provides HTTP routing, middleware, and OAuth handling for CLI and web interfaces.
//
// # Router Infrastructure
//
// The [Router] interface defines HTTP routing with middleware support.
//
// [Middleware] wraps handlers in reverse order (last added executes first), following the standard Go pattern.
//
// The [BasicRouter] implementation uses [http.ServeMux] internally with method filtering.
//
// # OAuth Callback Handler
//
// OAuthHandler implements the OAuth2 authorization code callback flow.
//
// The handler validates the state parameter (CSRF protection), exchanges the authorization code for tokens,
// and sends the result through a channel.
//
// It only processes one callback to prevent replay attacks.
//
// # Current Usage
//
// The server package currently supports CLI OAuth flows for Spotify authentication.
// When the user runs authentication commands, a temporary HTTP server starts on localhost:3000, handles the callback,
// and shuts down after receiving the OAuth token.
//
// # Web Application Integration
//
// The web package (internal/web) will extend this infrastructure with:
//   - Session middleware for persistent authentication state
//   - Playlist handlers rendering HTMX templates
//   - SSE streaming for real-time transfer progress
//   - Migration job management with repositories
//
// # Handler Interface
//
// Custom handlers implement the [Handler] interface, which wraps the stdlib handler interface and adds routes,
// allowing handlers to register multiple routes to encapsulate route definitions within the implementation.
package server
