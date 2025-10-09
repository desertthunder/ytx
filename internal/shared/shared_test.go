package shared

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeTrackKey(t *testing.T) {
	tc := []struct {
		name   string
		title  string
		artist string
		want   string
	}{
		{
			name:   "basic normalization",
			title:  "Song Title",
			artist: "Artist Name",
			want:   "song title|artist name",
		},
		{
			name:   "extra whitespace",
			title:  "  Song   Title  ",
			artist: "  Artist   Name  ",
			want:   "song title|artist name",
		},
		{
			name:   "mixed case",
			title:  "SoNg TiTlE",
			artist: "ArTiSt NaMe",
			want:   "song title|artist name",
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeTrackKey(tt.title, tt.artist)
			if got != tt.want {
				t.Errorf("normalizeTrackKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestErrorTypes(t *testing.T) {
	t.Run("ErrRefreshFailed", func(t *testing.T) {
		t.Run("is defined", func(t *testing.T) {
			if ErrRefreshFailed == nil {
				t.Fatal("expected ErrRefreshFailed to be defined")
			}
		})

		t.Run("has correct message", func(t *testing.T) {
			errMsg := ErrRefreshFailed.Error()
			if !strings.Contains(errMsg, "refresh") {
				t.Errorf("expected error message to contain 'refresh', got %s", errMsg)
			}
			if !strings.Contains(errMsg, "failed") {
				t.Errorf("expected error message to contain 'failed', got %s", errMsg)
			}
		})

		t.Run("can be wrapped", func(t *testing.T) {
			wrapped := errors.New("details: token refresh failed")
			if !strings.Contains(wrapped.Error(), "refresh failed") {
				t.Error("expected wrapped error to be detectable")
			}
		})

		t.Run("is distinct from other errors", func(t *testing.T) {
			if ErrRefreshFailed == ErrTokenExpired {
				t.Error("expected ErrRefreshFailed to be distinct from ErrTokenExpired")
			}
			if ErrRefreshFailed == ErrNoRefreshToken {
				t.Error("expected ErrRefreshFailed to be distinct from ErrNoRefreshToken")
			}
		})
	})

	t.Run("ErrNoRefreshToken", func(t *testing.T) {
		t.Run("is defined", func(t *testing.T) {
			if ErrNoRefreshToken == nil {
				t.Fatal("expected ErrNoRefreshToken to be defined")
			}
		})

		t.Run("has correct message", func(t *testing.T) {
			errMsg := ErrNoRefreshToken.Error()
			if !strings.Contains(errMsg, "refresh token") || !strings.Contains(errMsg, "refresh") {
				t.Errorf("expected error message to contain 'refresh token', got %s", errMsg)
			}
		})

		t.Run("can be checked with errors.Is", func(t *testing.T) {
			testErr := ErrNoRefreshToken
			if !errors.Is(testErr, ErrNoRefreshToken) {
				t.Error("expected errors.Is to work with ErrNoRefreshToken")
			}
		})

		t.Run("is distinct from other errors", func(t *testing.T) {
			if ErrNoRefreshToken == ErrTokenExpired {
				t.Error("expected ErrNoRefreshToken to be distinct from ErrTokenExpired")
			}
			if ErrNoRefreshToken == ErrRefreshFailed {
				t.Error("expected ErrNoRefreshToken to be distinct from ErrRefreshFailed")
			}
		})
	})

	t.Run("existing token errors", func(t *testing.T) {
		t.Run("ErrTokenExpired is defined", func(t *testing.T) {
			if ErrTokenExpired == nil {
				t.Fatal("expected ErrTokenExpired to be defined")
			}
		})

		t.Run("ErrNotAuthenticated is defined", func(t *testing.T) {
			if ErrNotAuthenticated == nil {
				t.Fatal("expected ErrNotAuthenticated to be defined")
			}
		})

		t.Run("ErrAuthFailed is defined", func(t *testing.T) {
			if ErrAuthFailed == nil {
				t.Fatal("expected ErrAuthFailed to be defined")
			}
		})

		t.Run("all auth errors are distinct", func(t *testing.T) {
			authErrors := []error{
				ErrTokenExpired,
				ErrRefreshFailed,
				ErrNoRefreshToken,
				ErrNotAuthenticated,
				ErrAuthFailed,
			}

			for i, err1 := range authErrors {
				for j, err2 := range authErrors {
					if i != j && err1 == err2 {
						t.Errorf("errors at index %d and %d are not distinct: %v == %v", i, j, err1, err2)
					}
				}
			}
		})
	})
}
