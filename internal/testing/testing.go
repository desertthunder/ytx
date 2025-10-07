// package testing contains shared testing utilities
package testing

import (
	"context"
	"errors"
	"io"
	"net/http"

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
