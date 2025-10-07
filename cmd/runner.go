package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/charmbracelet/log"
	"github.com/desertthunder/song-migrations/internal/services"
	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/desertthunder/song-migrations/internal/tasks"
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

// TODO: Refactor to extract setupLogic(configPath string) for better testability
func (r *Runner) Setup(ctx context.Context, cmd *cli.Command) error {
	configPath := cmd.String("config")

	var config *shared.Config
	if _, err := os.Stat(configPath); err == nil {
		if config, err = shared.LoadConfig(configPath); err != nil {
			r.logger.Warn("failed to load config, using defaults", "error", err)
			config = shared.DefaultConfig()
		}
	} else {
		r.logger.Info("config file not found, creating from template", "path", configPath)
		if err := shared.CreateConfigFile(configPath); err != nil {
			r.logger.Warn("failed to create config file, using defaults", "error", err)
			config = shared.DefaultConfig()
		} else {
			r.logger.Info("config file created", "path", configPath)
			if config, err = shared.LoadConfig(configPath); err != nil {
				r.logger.Warn("failed to load created config, using defaults", "error", err)
				config = shared.DefaultConfig()
			}
		}
	}

	r.logger.Info("initializing database", "path", config.Database.Path)

	db, err := shared.NewDatabase(config.Database.Path)
	if err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	defer db.Close()

	shared.ConfigureDatabase(db, config.Database.MaxOpenConns, config.Database.MaxIdleConns)

	r.logger.Info("running database migrations")
	if err := shared.RunMigrations(db); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	r.logger.Infof("setup complete for database: %v", config.Database.Path)
	return nil
}
