// package testing contains shared testing utilities
package testing

import (
	"context"
	"errors"
	"io"

	"github.com/desertthunder/song-migrations/internal/services"
)

// MockService is a test double for [services.Service]
type MockService struct{}

func (m *MockService) Authenticate(ctx context.Context, credentials map[string]string) error {
	return nil
}

func (m *MockService) GetPlaylists(ctx context.Context) ([]services.Playlist, error) {
	return []services.Playlist{}, nil
}
func (m *MockService) GetPlaylist(ctx context.Context, playlistID string) (*services.Playlist, error) {
	return nil, nil
}
func (m *MockService) ExportPlaylist(ctx context.Context, playlistID string) (*services.PlaylistExport, error) {
	return nil, nil
}
func (m *MockService) ImportPlaylist(ctx context.Context, playlist *services.PlaylistExport) (*services.Playlist, error) {
	return nil, nil
}
func (m *MockService) SearchTrack(ctx context.Context, title, artist string) (*services.Track, error) {
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
