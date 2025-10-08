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
					r.writePlainln("🔍 %s", update.Message)
				} else {
					r.writePlain("   %s\n", update.Message)
				}
			case tasks.CreatePlaylist:
				r.writePlainln("📝 %s", update.Message)
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
		r.writePlainln("Failed to match %d tracks:", result.FailedCount)
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

	r.logger.Infof("transfer diff requested source: %v dest %v", sourceID, destID)
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

	r.writePlainln("✓ Source: %s (%d tracks)", result.Comparison.SourcePlaylist.Playlist.Name, len(result.Comparison.SourcePlaylist.Tracks))
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
func (r *Runner) resolveService(name string) (services.Service, error) {
	switch name {
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
		return nil, fmt.Errorf("%w: invalid service '%s' (must be 'spotify' or 'youtube')", shared.ErrInvalidArgument, name)
	}
}
