// package shared defines shared helpers
package shared

import (
	"io"
	"os"

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
	opts := log.Options{ReportTimestamp: true, ReportCaller: true}
	return log.NewWithOptions(w, opts)
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
