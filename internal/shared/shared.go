// package shared defines shared helpers
package shared

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	"github.com/google/uuid"
)

// NewLogger creates a new [log.Logger] instance with the specified [io.Writer], with timestamps and caller reporting enabled.
//
// The writer defaults to [os.Stderr]
func NewLogger(w io.Writer) *log.Logger {
	if w == nil {
		w = os.Stderr
	}
	opts := log.Options{ReportTimestamp: true, ReportCaller: true, TimeFormat: time.Kitchen}
	return log.NewWithOptions(w, opts)
}

// NewFileLogger creates a new [log.Logger] that writes to a file at the given path.
//
// If the directory doesn't exist, it will be created. The logger uses the same
// timestamp and caller reporting settings as [NewLogger].
func NewFileLogger(path string) (*log.Logger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	opts := log.Options{ReportTimestamp: true, ReportCaller: true, TimeFormat: time.Kitchen, Level: log.DebugLevel}
	return log.NewWithOptions(file, opts), nil
}

// WithLogger creates a child [log.Logger] with the specified key-value pairs added to all log entries.
func WithLogger(l *log.Logger, kv ...any) *log.Logger {
	return l.With(kv...)
}

// SetLogLevel sets the [log.Level] for the given [log.Logger].
func SetLogLevel(l *log.Logger, ll log.Level) {
	l.SetLevel(ll)
}

// GenerateID generates a new v4 [uuid.UUID] as a string
func GenerateID() string {
	return uuid.New().String()
}

func MarshalJSON(data any, pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(data, "", "  ")
	}
	return json.Marshal(data)
}

// NormalizeTrackKey creates a normalized key for track comparison.
//
// Converts to lowercase and removes extra whitespace for fuzzy matching.
func NormalizeTrackKey(title, artist string) string {
	normalized := strings.ToLower(strings.TrimSpace(title)) + "|" + strings.ToLower(strings.TrimSpace(artist))
	return strings.Join(strings.Fields(normalized), " ")
}

// GenerateState generates a cryptographically secure random state token for CSRF protection.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// AbsolutePath converts a relative or absolute path to an absolute path.
func AbsolutePath(p string) (string, error) {
	if filepath.IsAbs(p) {
		return p, nil
	}
	return filepath.Abs(p)
}

// ExpandPath expands ~ to home directory in file paths.
func ExpandPath(p string) string {
	if p == "" {
		return p
	}

	if strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}

	return p
}

func ValidateJSON(data []byte) error {
	var jsonTest any
	if err := json.Unmarshal(data, &jsonTest); err != nil {
		return fmt.Errorf("%w: file is not valid JSON", ErrInvalidInput)
	} else {
		return nil
	}
}

func VerifyAndReadFile(p string) ([]byte, error) {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return []byte{}, fmt.Errorf("%w: file not found: %s", ErrInvalidArgument, p)
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read file: %w", err)
	}

	return data, nil
}
