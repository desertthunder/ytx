package main

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	"github.com/desertthunder/ytx/internal/tasks"
	"github.com/desertthunder/ytx/internal/ui"
	"github.com/urfave/cli/v3"
)

// TransferRun runs a full Spotify → YouTube Music sync.
func (r *Runner) TransferRun(ctx context.Context, cmd *cli.Command) error {
	sourceID := cmd.String("source")

	r.logger.Infof("starting transfer from source: %v", sourceID)

	r.writePlain("Starting playlist transfer...\n")
	r.writePlain("Source: %s\n\n", sourceID)

	progressCh := make(chan tasks.ProgressUpdate, 50)
	go func() {
		for update := range progressCh {
			switch update.Phase {
			case tasks.FetchSource:
				r.writePlain("📥 %s\n", update.Message)
			case tasks.SearchTracks:
				if update.Step == 0 {
					r.writePlain("\n🔍 %s\n", update.Message)
				} else {
					r.writePlain("   %s\n", update.Message)
				}
			case tasks.CreatePlaylist:
				r.writePlain("\n📝 %s\n", update.Message)
			}
		}
	}()

	result, err := r.engine.Run(ctx, sourceID, progressCh)
	close(progressCh)

	if err != nil {
		return err
	}

	r.writePlainHeader("Transfer Complete!")
	r.writePlain("Source: %s (%d tracks)\n", result.SourcePlaylist.Playlist.Name, result.TotalTracks)
	r.writePlain("Destination: %s (%d tracks)\n", result.DestPlaylist.Name, result.DestPlaylist.TrackCount)
	r.writePlain("Success rate: %d/%d (%.1f%%)\n", result.SuccessCount, result.TotalTracks, result.MatchPercentage)

	if result.FailedCount > 0 {
		r.writePlain("\nFailed to match %d tracks:\n", result.FailedCount)
		for _, match := range result.TrackMatches {
			if match.Error != nil {
				r.writePlain("  - %s - %s\n", match.Original.Artist, match.Original.Title)
			}
		}
	}

	return nil
}

// TransferDiff compares and shows missing tracks between two playlists.
func (r *Runner) TransferDiff(ctx context.Context, cmd *cli.Command) error {
	sourceID := cmd.String("source-id")
	destID := cmd.String("dest-id")
	sourceService := cmd.String("source-service")
	destService := cmd.String("dest-service")

	r.logger.Info("transfer diff requested", "source", sourceID, "dest", destID)
	r.writePlain("Comparing playlists...\n\n")

	srcService, err := r.resolveService(sourceService)
	if err != nil {
		return err
	}
	dstService, err := r.resolveService(destService)
	if err != nil {
		return err
	}

	progressCh := make(chan tasks.ProgressUpdate, 10)
	go func() {
		for update := range progressCh {
			r.writePlain("📥 %s\n", update.Message)
		}
	}()

	result, err := r.engine.Diff(ctx, srcService, dstService, sourceID, destID, progressCh)
	close(progressCh)

	if err != nil {
		return err
	}

	r.writePlain("\n✓ Source: %s (%d tracks)\n", result.Comparison.SourcePlaylist.Playlist.Name, len(result.Comparison.SourcePlaylist.Tracks))
	r.writePlain("✓ Destination: %s (%d tracks)\n\n", result.Comparison.DestPlaylist.Playlist.Name, len(result.Comparison.DestPlaylist.Tracks))

	r.writePlainHeader("Comparison Results")
	r.writePlain("Matched: %d tracks\n", result.Comparison.MatchedCount)
	r.writePlain("Missing from destination: %d tracks\n", len(result.Comparison.MissingInDest))
	r.writePlain("Extra in destination: %d tracks\n\n", len(result.Comparison.ExtraInDest))

	if len(result.Comparison.MissingInDest) > 0 {
		r.writePlain("Missing from destination:\n")
		for i, track := range result.Comparison.MissingInDest {
			r.writePlain("  %d. %s - %s", i+1, track.Artist, track.Title)
			if track.Album != "" {
				r.writePlain(" (%s)", track.Album)
			}
			r.writePlain("\n")
		}
		r.writePlain("\n")
	}

	if len(result.Comparison.ExtraInDest) > 0 {
		r.writePlain("Extra in destination (not in source):\n")
		for i, track := range result.Comparison.ExtraInDest {
			r.writePlain("  %d. %s - %s", i+1, track.Artist, track.Title)
			if track.Album != "" {
				r.writePlain(" (%s)", track.Album)
			}
			r.writePlain("\n")
		}
	}

	return nil
}

// TransferUI launches the interactive TUI for playlist transfer.
func (r *Runner) TransferUI(ctx context.Context, cmd *cli.Command) error {
	if r.spotify == nil {
		return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}
	if r.engine == nil {
		return fmt.Errorf("%w: transfer engine not initialized", shared.ErrServiceUnavailable)
	}

	model := ui.NewModel(ctx, r.spotify, r.engine)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running TUI: %w", err)
	}

	return nil
}

// resolveService resolves a service name to its corresponding Service instance.
func (r *Runner) resolveService(serviceName string) (services.Service, error) {
	switch serviceName {
	case "spotify":
		if r.spotify == nil {
			return nil, fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
		}
		return r.spotify, nil
	case "youtube", "ytmusic":
		if r.youtube == nil {
			return nil, fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
		}
		return r.youtube, nil
	default:
		return nil, fmt.Errorf("%w: invalid service '%s' (must be 'spotify' or 'youtube')", shared.ErrInvalidArgument, serviceName)
	}
}

// transferCommand handles playlist transfer operations (v0.6 stubs)
func transferCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:  "transfer",
		Usage: "Transfer playlists between services",
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Run full Spotify → YouTube Music sync",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "source",
						Usage:    "Source playlist name or ID",
						Required: true,
					},
				},
				Action: r.TransferRun,
			},
			{
				Name:  "ui",
				Usage: "Interactive TUI for playlist transfer",
				Action: r.TransferUI,
			},
			{
				Name:  "diff",
				Usage: "Compare and show missing tracks between two playlists",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "source-id",
						Usage:    "Source playlist ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "dest-id",
						Usage:    "Destination playlist ID",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "source-service",
						Usage:    "Source service (spotify or youtube)",
						Value:    "spotify",
						Required: false,
					},
					&cli.StringFlag{
						Name:     "dest-service",
						Usage:    "Destination service (spotify or youtube)",
						Value:    "youtube",
						Required: false,
					},
				},
				Action: r.TransferDiff,
			},
		},
	}
}
