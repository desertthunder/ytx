package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	"github.com/desertthunder/ytx/internal/tasks"
	"github.com/urfave/cli/v3"
	"golang.org/x/oauth2"
)

// Runner holds all dependencies for CLI commands and provides methods for each command action.
type Runner struct {
	config     *shared.Config
	configPath string
	spotify    services.Service
	youtube    services.Service
	api        *services.APIService
	httpClient *http.Client
	logger     *log.Logger
	output     io.Writer
	engine     *tasks.PlaylistEngine
}

// RunnerOpts contains configuration options for creating a Runner.
type RunnerOpts struct {
	Config     *shared.Config
	ConfigPath string
	Spotify    services.Service
	YouTube    services.Service
	API        *services.APIService
	HTTPClient *http.Client
	Logger     *log.Logger
	Output     io.Writer
}

// NewRunner creates a new Runner with the provided configuration
func NewRunner(opts RunnerOpts) *Runner {
	if opts.Config == nil {
		opts.Config = shared.DefaultConfig()
	}
	if opts.Logger == nil {
		opts.Logger = shared.NewLogger(nil)
	}
	if opts.Output == nil {
		opts.Output = os.Stdout
	}
	if opts.HTTPClient == nil {
		opts.HTTPClient = http.DefaultClient
	}

	engine := tasks.NewPlaylistEngine(opts.Spotify, opts.YouTube, opts.API)

	return &Runner{
		config:     opts.Config,
		configPath: opts.ConfigPath,
		spotify:    opts.Spotify,
		youtube:    opts.YouTube,
		api:        opts.API,
		httpClient: opts.HTTPClient,
		logger:     opts.Logger,
		output:     opts.Output,
		engine:     engine,
	}
}

func (r *Runner) register() []*cli.Command {
	commands := []*cli.Command{}
	for _, fn := range [](func(*Runner) *cli.Command){
		setupCommand, authCommand, spotifyCommand, apiCommand, ytmusicCommand, transferCommand, cacheCommand, tuiCommand,
	} {
		commands = append(commands, fn(r))
	}

	return commands
}

func (r *Runner) writeJSON(data any, pretty bool) error {
	var output []byte
	var err error

	if pretty {
		output, err = json.MarshalIndent(data, "", "  ")
	} else {
		output, err = json.Marshal(data)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if _, err := r.output.Write(output); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	if _, err := r.output.Write([]byte("\n")); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return nil
}

func (r *Runner) writePlain(format string, args ...any) error {
	text := fmt.Sprintf(format, args...)
	if _, err := r.output.Write([]byte(text)); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

func (r *Runner) writePlainln(format string, args ...any) error {
	text := "\n" + fmt.Sprintf(format, args...) + "\n"
	if _, err := r.output.Write([]byte(text)); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}
	return nil
}

func (r *Runner) writePlainHeader(title string) {
	r.writePlain("═══════════════════════════════════════\n")
	r.writePlain("%v\n", title)
	r.writePlain("═══════════════════════════════════════\n")
}

// SetLogger replaces the runner's logger with a new instance.
//
// This is useful for redirecting logs to a file when running the TUI.
func (r *Runner) SetLogger(logger *log.Logger) {
	r.logger = logger
}

// saveTokens updates the config with new tokens and persists to disk
func (r *Runner) saveTokens(token *oauth2.Token) error {
	if r.config == nil {
		return fmt.Errorf("config is nil")
	}

	if err := r.config.Credentials.Spotify.Update(token); err != nil {
		return fmt.Errorf("failed to update spotify configuration: %w", err)
	}

	if r.configPath != "" {
		if err := shared.SaveConfig(r.configPath, r.config); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		r.logger.Debugf("saved refreshed tokens to %s", r.configPath)
	}

	return nil
}
