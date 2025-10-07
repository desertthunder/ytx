package server

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"golang.org/x/oauth2"
)

// OAuthResult contains the result of an OAuth authorization flow.
type OAuthResult struct {
	Token *oauth2.Token
	err   error
}

func (o *OAuthResult) Error() error {
	return o.err
}

// OAuthHandler handles OAuth2 callback requests for authorization code flow.
// Implements the Handler interface for registration with a Router.
type OAuthHandler struct {
	config      *oauth2.Config
	state       string
	resultChan  chan OAuthResult
	once        sync.Once
	callbackHit bool
	mu          sync.Mutex
}

// NewOAuthHandler creates a new OAuth handler with the given OAuth2 config and state token.
// The state token should be cryptographically random for CSRF protection.
func NewOAuthHandler(config *oauth2.Config, state string) *OAuthHandler {
	return &OAuthHandler{
		config:     config,
		state:      state,
		resultChan: make(chan OAuthResult, 1),
	}
}

// Routes returns the HTTP routes this handler serves.
func (h *OAuthHandler) Routes() []string {
	return []string{"/callback"}
}

// ServeHTTP handles the OAuth callback request.
//
// Validates state parameter, exchanges authorization code for tokens, and sends the result through the result channel.
func (h *OAuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Only handle callback once
	h.mu.Lock()
	if h.callbackHit {
		h.mu.Unlock()
		http.Error(w, "Callback already processed", http.StatusBadRequest)
		return
	}
	h.callbackHit = true
	h.mu.Unlock()

	state := r.URL.Query().Get("state")
	if state != h.state {
		err := fmt.Errorf("invalid state parameter")
		h.Send(OAuthResult{err: err})
		http.Error(w, "Invalid state parameter", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		errParam := r.URL.Query().Get("error")
		errDesc := r.URL.Query().Get("error_description")
		err := fmt.Errorf("authorization failed: %s - %s", errParam, errDesc)
		h.Send(OAuthResult{err: err})
		http.Error(w, "Authorization failed", http.StatusBadRequest)
		return
	}

	token, err := h.config.Exchange(context.Background(), code)
	if err != nil {
		h.Send(OAuthResult{err: fmt.Errorf("token exchange failed: %w", err)})
		http.Error(w, "Token exchange failed", http.StatusInternalServerError)
		return
	}

	h.Send(OAuthResult{Token: token})

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Authorization Successful</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
               display: flex; align-items: center; justify-content: center; height: 100vh;
               margin: 0; background: #f5f5f5; }
        .container { text-align: center; background: white; padding: 2rem;
                     border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #1DB954; margin: 0 0 1rem 0; }
        p { color: #666; margin: 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>✓ Authorization Successful</h1>
        <p>You can close this window and return to the terminal.</p>
    </div>
</body>
</html>
`)
}

// Send sends the OAuth result through the channel (only once).
func (h *OAuthHandler) Send(result OAuthResult) {
	h.once.Do(func() {
		h.resultChan <- result
		close(h.resultChan)
	})
}

// Result returns the result channel for receiving OAuth flow completion.
//
// Channel will receive exactly one result and then be closed.
func (h *OAuthHandler) Result() <-chan OAuthResult {
	return h.resultChan
}
