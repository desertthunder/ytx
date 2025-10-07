// API service for making raw HTTP requests to the FastAPI proxy
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// APIService provides methods for making raw HTTP requests to the FastAPI proxy.
// FIXME: should implement [Service]
type APIService struct {
	baseURL    string
	httpClient *http.Client
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
