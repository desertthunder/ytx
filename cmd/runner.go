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
)

// Runner holds all dependencies for CLI commands and provides methods for each command action.
type Runner struct {
	config     *shared.Config
	spotify    services.Service
	youtube    services.Service
	api        *services.APIService
	httpClient *http.Client
	logger     *log.Logger
	output     io.Writer
	engine     *tasks.PlaylistEngine
}

// RunnerConfig contains configuration options for creating a Runner.
type RunnerConfig struct {
	Config     *shared.Config
	Spotify    services.Service
	YouTube    services.Service
	API        *services.APIService
	HTTPClient *http.Client
	Logger     *log.Logger
	Output     io.Writer
}

// NewRunner creates a new Runner with the provided configuration
func NewRunner(cfg RunnerConfig) *Runner {
	if cfg.Config == nil {
		cfg.Config = shared.DefaultConfig()
	}
	if cfg.Logger == nil {
		cfg.Logger = shared.NewLogger(nil)
	}
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}

	engine := tasks.NewPlaylistEngine(cfg.Spotify, cfg.YouTube, cfg.API)

	return &Runner{
		config:     cfg.Config,
		spotify:    cfg.Spotify,
		youtube:    cfg.YouTube,
		api:        cfg.API,
		httpClient: cfg.HTTPClient,
		logger:     cfg.Logger,
		output:     cfg.Output,
		engine:     engine,
	}
}

func (r *Runner) register() []*cli.Command {
	commands := []*cli.Command{}
	for _, fn := range [](func(*Runner) *cli.Command){
		setupCommand, authCommand, spotifyCommand, apiCommand, ytmusicCommand, transferCommand,
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
