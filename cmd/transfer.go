package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/desertthunder/song-migrations/internal/services"
	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/urfave/cli/v3"
)

// TransferRun runs a full Spotify → YouTube Music sync.
func (r *Runner) TransferRun(ctx context.Context, cmd *cli.Command) error {
	if r.spotify == nil {
		return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
	}
	if r.youtube == nil {
		return fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
	}

	sourceIDOrName := cmd.String("source")
	destName := cmd.String("dest")

	r.logger.Info("starting transfer", "source", sourceIDOrName, "dest", destName)
	r.writePlain("Starting playlist transfer...\n")
	r.writePlain("Source: %s\n", sourceIDOrName)
	r.writePlain("Destination: %s\n\n", destName)

	// Step 1: Get source playlist from Spotify
	r.writePlain("📥 Fetching source playlist from Spotify...\n")
	var sourcePlaylist *services.PlaylistExport
	var err error

	// Try as ID first, then search by name if that fails
	sourcePlaylist, err = r.spotify.ExportPlaylist(ctx, sourceIDOrName)
	if err != nil {
		r.logger.Info("source not found by ID, searching by name", "error", err)
		r.writePlain("   Source ID not found, searching by name...\n")

		playlists, err := r.spotify.GetPlaylists(ctx)
		if err != nil {
			return fmt.Errorf("%w: failed to get playlists: %v", shared.ErrAPIRequest, err)
		}

		var matchedID string
		for _, pl := range playlists {
			if pl.Name == sourceIDOrName {
				matchedID = pl.ID
				break
			}
		}

		if matchedID == "" {
			return fmt.Errorf("%w: no playlist found with name '%s'", shared.ErrPlaylistNotFound, sourceIDOrName)
		}

		sourcePlaylist, err = r.spotify.ExportPlaylist(ctx, matchedID)
		if err != nil {
			return fmt.Errorf("%w: failed to export playlist: %v", shared.ErrAPIRequest, err)
		}
	}

	r.writePlain("✓ Found playlist: %s (%d tracks)\n\n", sourcePlaylist.Playlist.Name, len(sourcePlaylist.Tracks))

	// Step 2: Search for each track on YouTube Music
	r.writePlain("🔍 Searching for tracks on YouTube Music...\n")

	type trackMatch struct {
		original services.Track
		matched  *services.Track
		err      error
	}

	matches := make([]trackMatch, len(sourcePlaylist.Tracks))
	successCount := 0

	for i, track := range sourcePlaylist.Tracks {
		r.writePlain("   [%d/%d] %s - %s", i+1, len(sourcePlaylist.Tracks), track.Artist, track.Title)

		ytTrack, err := r.youtube.SearchTrack(ctx, track.Title, track.Artist)
		matches[i] = trackMatch{
			original: track,
			matched:  ytTrack,
			err:      err,
		}

		if err != nil {
			r.writePlain(" ✗ Not found\n")
			r.logger.Warn("track not found", "title", track.Title, "artist", track.Artist, "error", err)
		} else {
			r.writePlain(" ✓\n")
			successCount++
		}
	}

	r.writePlain("\n✓ Matched %d/%d tracks\n\n", successCount, len(sourcePlaylist.Tracks))

	if successCount == 0 {
		return fmt.Errorf("no tracks were matched - cannot create empty playlist")
	}

	// Step 3: Create destination playlist on YouTube Music
	r.writePlain("📝 Creating playlist on YouTube Music...\n")

	matchedTracks := make([]services.Track, 0, successCount)
	for _, match := range matches {
		if match.matched != nil {
			matchedTracks = append(matchedTracks, *match.matched)
		}
	}

	destExport := &services.PlaylistExport{
		Playlist: services.Playlist{
			Name:        destName,
			Description: fmt.Sprintf("Migrated from Spotify: %s", sourcePlaylist.Playlist.Name),
			Public:      false,
		},
		Tracks: matchedTracks,
	}

	newPlaylist, err := r.youtube.ImportPlaylist(ctx, destExport)
	if err != nil {
		return fmt.Errorf("%w: failed to create playlist: %v", shared.ErrAPIRequest, err)
	}

	r.writePlain("✓ Playlist created: %s (ID: %s)\n\n", newPlaylist.Name, newPlaylist.ID)

	// Step 4: Summary
	r.writePlain("═══════════════════════════════════════\n")
	r.writePlain("Transfer Complete!\n")
	r.writePlain("═══════════════════════════════════════\n")
	r.writePlain("Source: %s (%d tracks)\n", sourcePlaylist.Playlist.Name, len(sourcePlaylist.Tracks))
	r.writePlain("Destination: %s (%d tracks)\n", newPlaylist.Name, newPlaylist.TrackCount)
	r.writePlain("Success rate: %d/%d (%.1f%%)\n", successCount, len(sourcePlaylist.Tracks),
		float64(successCount)/float64(len(sourcePlaylist.Tracks))*100)

	failedCount := len(sourcePlaylist.Tracks) - successCount
	if failedCount > 0 {
		r.writePlain("\nFailed to match %d tracks:\n", failedCount)
		for _, match := range matches {
			if match.err != nil {
				r.writePlain("  - %s - %s\n", match.original.Artist, match.original.Title)
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

	// Determine which services to use
	var sourceSvc, destSvc services.Service
	switch sourceService {
	case "spotify":
		if r.spotify == nil {
			return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
		}
		sourceSvc = r.spotify
	case "youtube", "ytmusic":
		if r.youtube == nil {
			return fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
		}
		sourceSvc = r.youtube
	default:
		return fmt.Errorf("%w: invalid source-service '%s' (must be 'spotify' or 'youtube')", shared.ErrInvalidArgument, sourceService)
	}

	switch destService {
	case "spotify":
		if r.spotify == nil {
			return fmt.Errorf("%w: Spotify service not initialized", shared.ErrServiceUnavailable)
		}
		destSvc = r.spotify
	case "youtube", "ytmusic":
		if r.youtube == nil {
			return fmt.Errorf("%w: YouTube Music service not initialized", shared.ErrServiceUnavailable)
		}
		destSvc = r.youtube
	default:
		return fmt.Errorf("%w: invalid dest-service '%s' (must be 'spotify' or 'youtube')", shared.ErrInvalidArgument, destService)
	}

	// Fetch both playlists
	r.writePlain("📥 Fetching source playlist (%s)...\n", sourceService)
	sourceExport, err := sourceSvc.ExportPlaylist(ctx, sourceID)
	if err != nil {
		return fmt.Errorf("%w: failed to export source playlist: %v", shared.ErrPlaylistNotFound, err)
	}

	r.writePlain("📥 Fetching destination playlist (%s)...\n", destService)
	destExport, err := destSvc.ExportPlaylist(ctx, destID)
	if err != nil {
		return fmt.Errorf("%w: failed to export destination playlist: %v", shared.ErrPlaylistNotFound, err)
	}

	r.writePlain("\n✓ Source: %s (%d tracks)\n", sourceExport.Playlist.Name, len(sourceExport.Tracks))
	r.writePlain("✓ Destination: %s (%d tracks)\n\n", destExport.Playlist.Name, len(destExport.Tracks))

	// Build destination track map for faster lookups
	destTrackMap := make(map[string]services.Track)
	destISRCMap := make(map[string]services.Track)

	for _, track := range destExport.Tracks {
		// Normalize for comparison
		normalizedKey := normalizeTrackKey(track.Title, track.Artist)
		destTrackMap[normalizedKey] = track

		// Also map by ISRC if available
		if track.ISRC != "" {
			destISRCMap[track.ISRC] = track
		}
	}

	// Find missing tracks (in source but not in dest)
	var missingInDest []services.Track
	var matchedCount int

	for _, srcTrack := range sourceExport.Tracks {
		matched := false

		// Try ISRC match first (most reliable)
		if srcTrack.ISRC != "" {
			if _, found := destISRCMap[srcTrack.ISRC]; found {
				matched = true
			}
		}

		// Fallback to title+artist match
		if !matched {
			normalizedKey := normalizeTrackKey(srcTrack.Title, srcTrack.Artist)
			if _, found := destTrackMap[normalizedKey]; found {
				matched = true
			}
		}

		if matched {
			matchedCount++
		} else {
			missingInDest = append(missingInDest, srcTrack)
		}
	}

	// Find extra tracks (in dest but not in source)
	sourceTrackMap := make(map[string]services.Track)
	sourceISRCMap := make(map[string]services.Track)

	for _, track := range sourceExport.Tracks {
		normalizedKey := normalizeTrackKey(track.Title, track.Artist)
		sourceTrackMap[normalizedKey] = track
		if track.ISRC != "" {
			sourceISRCMap[track.ISRC] = track
		}
	}

	var extraInDest []services.Track
	for _, destTrack := range destExport.Tracks {
		matched := false

		if destTrack.ISRC != "" {
			if _, found := sourceISRCMap[destTrack.ISRC]; found {
				matched = true
			}
		}

		if !matched {
			normalizedKey := normalizeTrackKey(destTrack.Title, destTrack.Artist)
			if _, found := sourceTrackMap[normalizedKey]; found {
				matched = true
			}
		}

		if !matched {
			extraInDest = append(extraInDest, destTrack)
		}
	}

	// Display results
	r.writePlain("═══════════════════════════════════════\n")
	r.writePlain("Comparison Results\n")
	r.writePlain("═══════════════════════════════════════\n")
	r.writePlain("Matched: %d tracks\n", matchedCount)
	r.writePlain("Missing from destination: %d tracks\n", len(missingInDest))
	r.writePlain("Extra in destination: %d tracks\n\n", len(extraInDest))

	if len(missingInDest) > 0 {
		r.writePlain("Missing from destination:\n")
		for i, track := range missingInDest {
			r.writePlain("  %d. %s - %s", i+1, track.Artist, track.Title)
			if track.Album != "" {
				r.writePlain(" (%s)", track.Album)
			}
			r.writePlain("\n")
		}
		r.writePlain("\n")
	}

	if len(extraInDest) > 0 {
		r.writePlain("Extra in destination (not in source):\n")
		for i, track := range extraInDest {
			r.writePlain("  %d. %s - %s", i+1, track.Artist, track.Title)
			if track.Album != "" {
				r.writePlain(" (%s)", track.Album)
			}
			r.writePlain("\n")
		}
	}

	return nil
}

// normalizeTrackKey creates a normalized key for track comparison.
// Converts to lowercase and removes extra whitespace for fuzzy matching.
func normalizeTrackKey(title, artist string) string {
	normalized := strings.ToLower(strings.TrimSpace(title)) + "|" + strings.ToLower(strings.TrimSpace(artist))
	return strings.Join(strings.Fields(normalized), " ")
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
					&cli.StringFlag{
						Name:     "dest",
						Usage:    "Destination playlist name",
						Required: true,
					},
				},
				Action: r.TransferRun,
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
