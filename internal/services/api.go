// API service for making raw HTTP requests to the FastAPI proxy
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// APIService provides methods for making raw HTTP requests to the FastAPI proxy.
// FIXME: should implement [Service]
type APIService struct {
	baseURL    string
	httpClient *http.Client
	authData   string // JSON string of auth headers
}

// NewAPIService creates a new API service instance for the FastAPI proxy.
func NewAPIService(baseURL string, client *http.Client) *APIService {
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}
	if client == nil {
		client = http.DefaultClient
	}

	return &APIService{
		baseURL:    baseURL,
		httpClient: client,
	}
}

// SetAuthFile reads a JSON authentication file and stores its JSON data for subsequent requests.
//
// The auth data is sent to the proxy via X-Auth-Data header (minified to avoid newlines).
func (a *APIService) SetAuthFile(authFile string) error {
	if authFile == "" {
		a.authData = ""
		return nil
	}

	authBytes, err := os.ReadFile(authFile)
	if err != nil {
		return fmt.Errorf("failed to read auth file: %w", err)
	}

	var authObj map[string]any
	if err := json.Unmarshal(authBytes, &authObj); err != nil {
		return fmt.Errorf("auth file contains invalid JSON: %w", err)
	}

	minifiedBytes, err := json.Marshal(authObj)
	if err != nil {
		return fmt.Errorf("failed to minify auth JSON: %w", err)
	}

	a.authData = string(minifiedBytes)
	return nil
}

// APIResponse represents a raw API response with status and body.
type APIResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	IsJSON     bool
	JSONData   any
}

// Get performs a GET request to the specified path and returns the raw response.
func (a *APIService) Get(ctx context.Context, path string) (*APIResponse, error) {
	fullURL := a.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if a.authData != "" {
		req.Header.Set("X-Auth-Data", a.authData)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}

	var jsonData any
	if err := json.Unmarshal(body, &jsonData); err == nil {
		apiResp.IsJSON = true
		apiResp.JSONData = jsonData
	}

	return apiResp, nil
}

// Post performs a POST request with the given JSON data and returns the raw response.
func (a *APIService) Post(ctx context.Context, path string, data []byte) (*APIResponse, error) {
	fullURL := a.baseURL + path

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fullURL, bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if a.authData != "" {
		req.Header.Set("X-Auth-Data", a.authData)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	apiResp := &APIResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       body,
	}

	var jsonData any
	if err := json.Unmarshal(body, &jsonData); err == nil {
		apiResp.IsJSON = true
		apiResp.JSONData = jsonData
	}

	return apiResp, nil
}

// UploadJSON uploads JSON data to the specified path.
func (a *APIService) UploadJSON(ctx context.Context, path string, jsonData []byte) (*APIResponse, error) {
	return a.Post(ctx, path, jsonData)
}

// BrowserSetupRequest represents the request body for browser authentication setup.
type BrowserSetupRequest struct {
	HeadersRaw string `json:"headers_raw"`
	Filepath   string `json:"filepath,omitempty"`
}

// BrowserSetupResponse represents the response from the setup endpoint.
type BrowserSetupResponse struct {
	Success     bool           `json:"success"`
	Filepath    string         `json:"filepath"`
	Message     string         `json:"message"`
	AuthContent map[string]any `json:"auth_content"`
}

// SetupBrowser configures browser authentication by sending headers_raw to the proxy.
//
// The proxy generates browser.json format and returns the auth content.
func (a *APIService) SetupBrowser(ctx context.Context, raw string) (*BrowserSetupResponse, error) {
	req := BrowserSetupRequest{HeadersRaw: raw}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := a.Post(ctx, "/api/setup", reqBytes)
	if err != nil {
		return nil, fmt.Errorf("setup request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("setup failed with status %d: %s", resp.StatusCode, string(resp.Body))
	}

	var setupResp BrowserSetupResponse
	if err := json.Unmarshal(resp.Body, &setupResp); err != nil {
		return nil, fmt.Errorf("failed to parse setup response: %w", err)
	}

	return &setupResp, nil
}
