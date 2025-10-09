package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/desertthunder/ytx/internal/shared"
	"github.com/desertthunder/ytx/internal/ui"
	"github.com/urfave/cli/v3"
)

// TUI launches the interactive terminal UI for playlist transfer.
func (r *Runner) TUI(ctx context.Context, cmd *cli.Command) error {
	if r.spotify == nil {
		return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}
	if r.engine == nil {
		return fmt.Errorf("%w: transfer engine not initialized", shared.ErrServiceUnavailable)
	}

	// Redirect logs to file to avoid interfering with TUI rendering
	fileLogger, err := shared.NewFileLogger("./tmp/ytx-tui.log")
	if err != nil {
		return fmt.Errorf("failed to create file logger: %w", err)
	}
	r.SetLogger(fileLogger)

	model := ui.NewModel(ctx, r.spotify, r.engine)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}
