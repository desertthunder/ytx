// package testing contains shared testing utilities
package testing

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/desertthunder/ytx/internal/models"
)

// MockService is a test double for [services.Service]
type MockService struct{}

func (m *MockService) Authenticate(ctx context.Context, credentials map[string]string) error {
	return nil
}

func (m *MockService) GetPlaylists(ctx context.Context) ([]models.Playlist, error) {
	return []models.Playlist{}, nil
}
func (m *MockService) GetPlaylist(ctx context.Context, playlistID string) (*models.Playlist, error) {
	return nil, nil
}
func (m *MockService) ExportPlaylist(ctx context.Context, playlistID string) (*models.PlaylistExport, error) {
	return nil, nil
}
func (m *MockService) ImportPlaylist(ctx context.Context, playlist *models.PlaylistExport) (*models.Playlist, error) {
	return nil, nil
}
func (m *MockService) SearchTrack(ctx context.Context, title, artist string) (*models.Track, error) {
	return nil, nil
}
func (m *MockService) Name() string { return "mock" }

// FWriter always returns an error on Write
type FWriter struct{}

func (f *FWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write failed")
}

// LimitedWriter fails after a certain number of writes
type LimitedWriter struct {
	maxWrites int
	written   int
	target    io.Writer
}

func (l *LimitedWriter) Write(p []byte) (n int, err error) {
	if l.written >= l.maxWrites {
		return 0, errors.New("write limit exceeded")
	}
	l.written++
	return l.target.Write(p)
}

func NewLimitedWriter(maxWrites, written int, target io.Writer) LimitedWriter {
	return LimitedWriter{maxWrites: maxWrites, written: written, target: target}
}

// MockRoundTripper allows custom HTTP responses for testing
type MockRoundTripper struct {
	response *http.Response
	err      error
}

func NewMockRoundTripper(r *http.Response, e error) *MockRoundTripper {
	return &MockRoundTripper{response: r, err: e}
}

func (m *MockRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return m.response, m.err
}

// FCloser simulates a failure when reading response body
type FCloser struct{}

func (f *FCloser) Read(p []byte) (n int, err error) {
	return 0, errors.New("read failed")
}

func (f *FCloser) Close() error {
	return nil
}

func MustGetwd(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	return wd
}

func MustChdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Failed to change directory to %s: %v", dir, err)
	}
}

func AssertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("File does not exist: %s", path)
	}
}

func AssertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		t.Errorf("Directory does not exist: %s", path)
		return
	}
	if !info.IsDir() {
		t.Errorf("Path is not a directory: %s", path)
	}
}

func MustReadFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read file %s: %v", path, err)
	}
	return string(content)
}
