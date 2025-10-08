package shared

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCurlCommand(t *testing.T) {
	tt := []struct {
		name        string
		curlCmd     string
		wantHeaders map[string]string
		wantCookie  string
		wantErr     bool
	}{
		{
			name:    "single header with single quotes",
			curlCmd: `curl -H 'Authorization: Bearer token123' https://api.example.com`,
			wantHeaders: map[string]string{
				"Authorization": "Bearer token123",
			},
			wantCookie: "",
			wantErr:    false,
		},
		{
			name:    "single header with double quotes",
			curlCmd: `curl -H "Authorization: Bearer token123" https://api.example.com`,
			wantHeaders: map[string]string{
				"Authorization": "Bearer token123",
			},
			wantCookie: "",
			wantErr:    false,
		},
		{
			name:    "multiple headers",
			curlCmd: `curl -H 'Content-Type: application/json' -H 'Authorization: Bearer token' https://api.example.com`,
			wantHeaders: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token",
			},
			wantCookie: "",
			wantErr:    false,
		},
		{
			name:        "cookie in -b flag with single quotes",
			curlCmd:     `curl -b 'session=abc123' https://api.example.com`,
			wantHeaders: map[string]string{},
			wantCookie:  "session=abc123",
			wantErr:     false,
		},
		{
			name:        "cookie in -b flag with double quotes",
			curlCmd:     `curl -b "session=abc123" https://api.example.com`,
			wantHeaders: map[string]string{},
			wantCookie:  "session=abc123",
			wantErr:     false,
		},
		{
			name:        "cookie in -H header",
			curlCmd:     `curl -H 'Cookie: session=abc123; token=xyz' https://api.example.com`,
			wantHeaders: map[string]string{},
			wantCookie:  "session=abc123; token=xyz",
			wantErr:     false,
		},
		{
			name:    "cookie header is excluded from regular headers",
			curlCmd: `curl -H 'Cookie: session=abc123' -H 'Authorization: Bearer token' https://api.example.com`,
			wantHeaders: map[string]string{
				"Authorization": "Bearer token",
			},
			wantCookie: "session=abc123",
			wantErr:    false,
		},
		{
			name: "multiline curl with backslashes",
			curlCmd: `curl -H 'Authorization: Bearer token' \
-H 'Content-Type: application/json' \
https://api.example.com`,
			wantHeaders: map[string]string{
				"Authorization": "Bearer token",
				"Content-Type":  "application/json",
			},
			wantCookie: "",
			wantErr:    false,
		},
		{
			name:    "headers with spaces around colon",
			curlCmd: `curl -H 'Authorization : Bearer token' https://api.example.com`,
			wantHeaders: map[string]string{
				"Authorization": "Bearer token",
			},
			wantCookie: "",
			wantErr:    false,
		},
		{
			name:        "-b cookie takes precedence over -H cookie",
			curlCmd:     `curl -H 'Cookie: old=value' -b 'new=value' https://api.example.com`,
			wantHeaders: map[string]string{},
			wantCookie:  "new=value",
			wantErr:     false,
		},
		{
			name:    "no headers or cookies",
			curlCmd: `curl https://api.example.com`,
			wantErr: true,
		},
		{
			name:    "empty command",
			curlCmd: "",
			wantErr: true,
		},
		{
			name: "complex real-world example",
			curlCmd: `curl 'https://music.youtube.com/api' \
  -H 'accept: */*' \
  -H 'accept-language: en-US,en;q=0.9' \
  -H 'authorization: SAPISIDHASH token_here' \
  -H 'content-type: application/json' \
  -H 'cookie: VISITOR_INFO=xyz; CONSENT=YES' \
  --data-raw '{"context":{}}'`,
			wantHeaders: map[string]string{
				"accept":          "*/*",
				"accept-language": "en-US,en;q=0.9",
				"authorization":   "SAPISIDHASH token_here",
				"content-type":    "application/json",
			},
			wantCookie: "VISITOR_INFO=xyz; CONSENT=YES",
			wantErr:    false,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseCurlCommand(tc.curlCmd)

			if (err != nil) != tc.wantErr {
				t.Errorf("ParseCurlCommand() error = %v, wantErr %v", err, tc.wantErr)
				return
			}

			if tc.wantErr {
				return
			}

			if result == nil {
				t.Fatal("ParseCurlCommand() returned nil result")
			}

			if len(result.Headers) != len(tc.wantHeaders) {
				t.Errorf("ParseCurlCommand() headers count = %v, want %v", len(result.Headers), len(tc.wantHeaders))
			}

			for key, want := range tc.wantHeaders {
				if got := result.Headers[key]; got != want {
					t.Errorf("ParseCurlCommand() header[%s] = %v, want %v", key, got, want)
				}
			}

			if result.Cookie != tc.wantCookie {
				t.Errorf("ParseCurlCommand() cookie = %v, want %v", result.Cookie, tc.wantCookie)
			}
		})
	}
}

func TestParseCurlFile(t *testing.T) {
	t.Run("successful file parse", func(t *testing.T) {
		tmpDir := t.TempDir()
		curlFile := filepath.Join(tmpDir, "curl.sh")

		curlCmd := `curl -H 'Authorization: Bearer token123' -H 'Content-Type: application/json' https://api.example.com`
		if err := os.WriteFile(curlFile, []byte(curlCmd), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		result, err := ParseCurlFile(curlFile)
		if err != nil {
			t.Fatalf("ParseCurlFile() error = %v", err)
		}

		if len(result.Headers) != 2 {
			t.Errorf("ParseCurlFile() headers count = %v, want 2", len(result.Headers))
		}

		if result.Headers["Authorization"] != "Bearer token123" {
			t.Errorf("ParseCurlFile() Authorization = %v, want %v", result.Headers["Authorization"], "Bearer token123")
		}
	})

	t.Run("file does not exist", func(t *testing.T) {
		_, err := ParseCurlFile("/nonexistent/file.sh")
		if err == nil {
			t.Error("ParseCurlFile() expected error for nonexistent file")
		}
	})

	t.Run("file with no valid headers", func(t *testing.T) {
		tmpDir := t.TempDir()
		curlFile := filepath.Join(tmpDir, "invalid.sh")

		if err := os.WriteFile(curlFile, []byte("curl https://example.com"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		_, err := ParseCurlFile(curlFile)
		if err == nil {
			t.Error("ParseCurlFile() expected error for file with no headers")
		}
	})
}

func TestCurlHeaders_ToHeadersRaw(t *testing.T) {
	tests := []struct {
		name    string
		headers *CurlHeaders
		want    string
	}{
		{
			name: "headers only",
			headers: &CurlHeaders{
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token",
				},
				Cookie: "",
			},
			// Note: map iteration order is not guaranteed, so we need to check both possibilities
		},
		{
			name: "cookie only",
			headers: &CurlHeaders{
				Headers: map[string]string{},
				Cookie:  "session=abc123",
			},
			want: "cookie: session=abc123",
		},
		{
			name: "headers and cookie",
			headers: &CurlHeaders{
				Headers: map[string]string{
					"Authorization": "Bearer token",
				},
				Cookie: "session=abc123",
			},
		},
		{
			name: "empty headers",
			headers: &CurlHeaders{
				Headers: map[string]string{},
				Cookie:  "",
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.headers.ToHeadersRaw()

			if tt.name == "cookie only" || tt.name == "empty headers" {
				if got != tt.want {
					t.Errorf("ToHeadersRaw() = %v, want %v", got, tt.want)
				}
				return
			}

			// For cases with headers, verify the output contains expected parts
			if tt.name == "headers only" {
				if !strings.Contains(got, "Content-Type: application/json") {
					t.Errorf("ToHeadersRaw() missing Content-Type header")
				}
				if !strings.Contains(got, "Authorization: Bearer token") {
					t.Errorf("ToHeadersRaw() missing Authorization header")
				}
				if strings.Contains(got, "cookie:") {
					t.Errorf("ToHeadersRaw() should not contain cookie line")
				}
			}

			if tt.name == "headers and cookie" {
				if !strings.Contains(got, "Authorization: Bearer token") {
					t.Errorf("ToHeadersRaw() missing Authorization header")
				}
				if !strings.Contains(got, "cookie: session=abc123") {
					t.Errorf("ToHeadersRaw() missing cookie line")
				}
			}
		})
	}
}
