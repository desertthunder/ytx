package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tu "github.com/desertthunder/ytx/internal/testing"
)

func TestAPIService(t *testing.T) {
	t.Run("New", func(t *testing.T) {
		t.Run("With Custom BaseURL and Client", func(t *testing.T) {
			customClient := &http.Client{}
			srv := NewAPIService("http://example.com", customClient)

			if srv.baseURL != "http://example.com" {
				t.Errorf("expected baseURL 'http://example.com', got %s", srv.baseURL)
			}
			if srv.httpClient != customClient {
				t.Error("expected custom client to be used")
			}
		})

		t.Run("With Empty BaseURL", func(t *testing.T) {
			srv := NewAPIService("", nil)

			if srv.baseURL != "http://localhost:8080" {
				t.Errorf("expected default baseURL 'http://localhost:8080', got %s", srv.baseURL)
			}
		})

		t.Run("With Nil Client", func(t *testing.T) {
			srv := NewAPIService("http://example.com", nil)

			if srv.httpClient != http.DefaultClient {
				t.Error("expected http.DefaultClient to be used")
			}
		})
	})

	t.Run("Get", func(t *testing.T) {
		t.Run("Successful Request With JSON Response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET method, got %s", r.Method)
				}
				if r.URL.Path != "/test" {
					t.Errorf("expected path '/test', got %s", r.URL.Path)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"status": "success"})
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			resp, err := srv.Get(context.Background(), "/test")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
			if !resp.IsJSON {
				t.Error("expected response to be JSON")
			}
			if resp.JSONData == nil {
				t.Error("expected JSONData to be populated")
			}
		})

		t.Run("Successful Request With Non-JSON Response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("plain text response"))
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			resp, err := srv.Get(context.Background(), "/test")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.IsJSON {
				t.Error("expected response to not be JSON")
			}
			if resp.JSONData != nil {
				t.Error("expected JSONData to be nil")
			}
			if string(resp.Body) != "plain text response" {
				t.Errorf("expected body 'plain text response', got %s", string(resp.Body))
			}
		})

		t.Run("Failed Request Creation", func(t *testing.T) {
			srv := NewAPIService("http://example.com", nil)
			_, err := srv.Get(context.Background(), "/test\x00invalid")

			if err == nil {
				t.Error("expected error for invalid URL")
			}
			if !strings.Contains(err.Error(), "failed to create request") {
				t.Errorf("expected 'failed to create request' error, got %v", err)
			}
		})

		t.Run("Failed HTTP Request", func(t *testing.T) {
			client := &http.Client{
				Transport: tu.NewMockRoundTripper(nil, errors.New("connection failed")),
			}

			srv := NewAPIService("http://example.com", client)
			_, err := srv.Get(context.Background(), "/test")

			if err == nil {
				t.Error("expected error for failed request")
			}
			if !strings.Contains(err.Error(), "request failed") {
				t.Errorf("expected 'request failed' error, got %v", err)
			}
		})

		t.Run("Failed Response Body Read", func(t *testing.T) {
			client := &http.Client{
				Transport: tu.NewMockRoundTripper(&http.Response{
					StatusCode: http.StatusOK,
					Body:       &tu.FCloser{},
					Header:     http.Header{},
				}, nil),
			}

			srv := NewAPIService("http://example.com", client)
			_, err := srv.Get(context.Background(), "/test")

			if err == nil {
				t.Error("expected error for failed body read")
			}
			if !strings.Contains(err.Error(), "failed to read response") {
				t.Errorf("expected 'failed to read response' error, got %v", err)
			}
		})

		t.Run("With Canceled Context", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			srv := NewAPIService(server.URL, nil)
			_, err := srv.Get(ctx, "/test")

			if err == nil {
				t.Error("expected error for canceled context")
			}
		})

		t.Run("Response Headers Are Preserved", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Custom-Header", "test-value")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("test"))
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			resp, err := srv.Get(context.Background(), "/test")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.Headers.Get("X-Custom-Header") != "test-value" {
				t.Errorf("expected custom header 'test-value', got %s", resp.Headers.Get("X-Custom-Header"))
			}
		})
	})

	t.Run("Post", func(t *testing.T) {
		t.Run("Successful Request With JSON Response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST method, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type 'application/json', got %s", r.Header.Get("Content-Type"))
				}

				body, _ := io.ReadAll(r.Body)
				var data map[string]string
				if err := json.Unmarshal(body, &data); err != nil {
					t.Errorf("failed to unmarshal request body: %v", err)
				}
				if data["test"] != "data" {
					t.Errorf("expected request data 'test:data', got %v", data)
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]string{"id": "123"})
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			requestData, _ := json.Marshal(map[string]string{"test": "data"})
			resp, err := srv.Post(context.Background(), "/test", requestData)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Errorf("expected status 201, got %d", resp.StatusCode)
			}
			if !resp.IsJSON {
				t.Error("expected response to be JSON")
			}
		})

		t.Run("Successful Request With Non-JSON Response", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("created"))
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			resp, err := srv.Post(context.Background(), "/test", []byte("data"))

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.IsJSON {
				t.Error("expected response to not be JSON")
			}
			if string(resp.Body) != "created" {
				t.Errorf("expected body 'created', got %s", string(resp.Body))
			}
		})

		t.Run("Failed Request Creation", func(t *testing.T) {
			srv := NewAPIService("http://example.com", nil)
			_, err := srv.Post(context.Background(), "/test\x00invalid", []byte("data"))

			if err == nil {
				t.Error("expected error for invalid URL")
			}
			if !strings.Contains(err.Error(), "failed to create request") {
				t.Errorf("expected 'failed to create request' error, got %v", err)
			}
		})

		t.Run("Failed HTTP Request", func(t *testing.T) {
			client := &http.Client{
				Transport: tu.NewMockRoundTripper(nil, errors.New("connection failed")),
			}

			srv := NewAPIService("http://example.com", client)
			_, err := srv.Post(context.Background(), "/test", []byte("data"))

			if err == nil {
				t.Error("expected error for failed request")
			}
			if !strings.Contains(err.Error(), "request failed") {
				t.Errorf("expected 'request failed' error, got %v", err)
			}
		})

		t.Run("Failed Response Body Read", func(t *testing.T) {
			client := &http.Client{
				Transport: tu.NewMockRoundTripper(&http.Response{
					StatusCode: http.StatusOK,
					Body:       &tu.FCloser{},
					Header:     http.Header{},
				}, nil),
			}

			srv := NewAPIService("http://example.com", client)
			_, err := srv.Post(context.Background(), "/test", []byte("data"))

			if err == nil {
				t.Error("expected error for failed body read")
			}
			if !strings.Contains(err.Error(), "failed to read response") {
				t.Errorf("expected 'failed to read response' error, got %v", err)
			}
		})

		t.Run("With Canceled Context", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			ctx, cancel := context.WithCancel(context.Background())
			cancel()

			srv := NewAPIService(server.URL, nil)
			_, err := srv.Post(ctx, "/test", []byte("data"))

			if err == nil {
				t.Error("expected error for canceled context")
			}
		})

		t.Run("Empty Request Body", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if len(body) != 0 {
					t.Errorf("expected empty body, got %d bytes", len(body))
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			_, err := srv.Post(context.Background(), "/test", []byte{})

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	})

	t.Run("UploadJSON", func(t *testing.T) {
		t.Run("Calls Post Method", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST method, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("expected Content-Type 'application/json', got %s", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"uploaded": true}`))
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			jsonData, _ := json.Marshal(map[string]any{"key": "value"})
			resp, err := srv.UploadJSON(context.Background(), "/upload", jsonData)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", resp.StatusCode)
			}
		})
	})

	t.Run("APIResponse", func(t *testing.T) {
		t.Run("JSON Detection", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"valid": "json"}`))
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			resp, err := srv.Get(context.Background(), "/test")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if !resp.IsJSON {
				t.Error("expected valid JSON to be detected")
			}

			jsonMap, ok := resp.JSONData.(map[string]any)
			if !ok {
				t.Error("expected JSONData to be map[string]interface{}")
			}
			if jsonMap["valid"] != "json" {
				t.Errorf("expected JSONData['valid'] to be 'json', got %v", jsonMap["valid"])
			}
		})

		t.Run("Invalid JSON Detection", func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("not json"))
			}))
			defer server.Close()

			srv := NewAPIService(server.URL, nil)
			resp, err := srv.Get(context.Background(), "/test")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if resp.IsJSON {
				t.Error("expected invalid JSON to not be detected as JSON")
			}
			if resp.JSONData != nil {
				t.Error("expected JSONData to be nil for invalid JSON")
			}
		})
	})
}
