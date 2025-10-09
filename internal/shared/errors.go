package shared

import "fmt"

var (
	ErrNotImplemented = fmt.Errorf("not implemented")

	// Configuration errors
	ErrMissingConfig      = fmt.Errorf("configuration not found")
	ErrInvalidConfig      = fmt.Errorf("invalid configuration")
	ErrMissingCredentials = fmt.Errorf("missing credentials")
	ErrInvalidCredentials = fmt.Errorf("invalid credentials")

	// Authentication errors
	ErrAuthFailed       = fmt.Errorf("authentication failed")
	ErrNotAuthenticated = fmt.Errorf("not authenticated")
	ErrTokenExpired     = fmt.Errorf("access token expired")
	ErrRefreshFailed    = fmt.Errorf("token refresh failed")
	ErrNoRefreshToken   = fmt.Errorf("no refresh token available")
	ErrTimeout          = fmt.Errorf("operation timed out")

	// API and service errors
	ErrAPIRequest         = fmt.Errorf("API request failed")
	ErrServiceUnavailable = fmt.Errorf("service unavailable")
	ErrPlaylistNotFound   = fmt.Errorf("playlist not found")
	ErrTrackNotFound      = fmt.Errorf("track not found")

	// Input validation errors
	ErrInvalidInput    = fmt.Errorf("invalid input")
	ErrMissingArgument = fmt.Errorf("missing required argument")
	ErrInvalidArgument = fmt.Errorf("invalid argument")
	ErrInvalidFlag     = fmt.Errorf("invalid flag value")
)
