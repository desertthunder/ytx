package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/urfave/cli/v3"
)

// APIGet makes a direct GET request to the proxy
func (r *Runner) APIGet(ctx context.Context, cmd *cli.Command) error {
	path := cmd.StringArg("path")
	useJSON := cmd.Bool("json")

	r.logger.Info("GET request", "path", path)

	resp, err := r.api.Get(ctx, path)
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d, body: %s", shared.ErrAPIRequest, resp.StatusCode, string(resp.Body))
	}

	if useJSON {
		if resp.IsJSON {
			return r.writeJSON(resp.JSONData, false)
		}
		r.output.Write(resp.Body)
		r.output.Write([]byte("\n"))
		return nil
	}

	if resp.IsJSON {
		return r.writeJSON(resp.JSONData, true)
	}

	r.output.Write(resp.Body)
	r.output.Write([]byte("\n"))
	return nil
}

// APIPost makes a direct POST request to the proxy
func (r *Runner) APIPost(ctx context.Context, cmd *cli.Command) error {
	path := cmd.StringArg("path")
	data := cmd.String("data")

	if data == "" {
		return fmt.Errorf("%w: --data flag is required", shared.ErrMissingArgument)
	}

	r.logger.Info("POST request", "path", path)

	var jsonTest any
	if err := json.Unmarshal([]byte(data), &jsonTest); err != nil {
		return fmt.Errorf("%w: data is not valid JSON: %v", shared.ErrInvalidInput, err)
	}

	resp, err := r.api.Post(ctx, path, []byte(data))
	if err != nil {
		return fmt.Errorf("%w: %v", shared.ErrAPIRequest, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: status %d, body: %s", shared.ErrAPIRequest, resp.StatusCode, string(resp.Body))
	}

	if resp.IsJSON {
		return r.writeJSON(resp.JSONData, true)
	}

	r.output.Write(resp.Body)
	r.output.Write([]byte("\n"))
	return nil
}

// APIDump fetches and displays the full proxy state.
func (r *Runner) APIDump(ctx context.Context, cmd *cli.Command) error {
	pretty := cmd.Bool("pretty")
	save := cmd.Bool("save")

	r.logger.Info("dumping API state")
	r.writePlain("Fetching proxy state...\n\n")

	type DumpData struct {
		Health         any   `json:"health"`
		Playlists      any   `json:"playlists,omitempty"`
		Songs          any   `json:"songs,omitempty"`
		Albums         any   `json:"albums,omitempty"`
		Artists        any   `json:"artists,omitempty"`
		LikedSongs     any   `json:"liked_songs,omitempty"`
		History        any   `json:"history,omitempty"`
		UploadedSongs  any   `json:"uploaded_songs,omitempty"`
		UploadedAlbums any   `json:"uploaded_albums,omitempty"`
		Errors         []any `json:"errors,omitempty"`
	}

	dump := DumpData{
		Errors: []any{},
	}

	// Fetch health
	r.writePlain("📊 Fetching health status...\n")
	if resp, err := r.api.Get(ctx, "/health"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.Health = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/health", "error": err.Error()})
		r.logger.Warn("failed to fetch health", "error", err)
	}

	// Fetch library playlists
	r.writePlain("📝 Fetching playlists...\n")
	if resp, err := r.api.Get(ctx, "/api/library/playlists"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.Playlists = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/library/playlists", "error": err.Error()})
		r.logger.Warn("failed to fetch playlists", "error", err)
	}

	// Fetch library songs
	r.writePlain("🎵 Fetching songs...\n")
	if resp, err := r.api.Get(ctx, "/api/library/songs"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.Songs = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/library/songs", "error": err.Error()})
		r.logger.Warn("failed to fetch songs", "error", err)
	}

	// Fetch library albums
	r.writePlain("💿 Fetching albums...\n")
	if resp, err := r.api.Get(ctx, "/api/library/albums"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.Albums = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/library/albums", "error": err.Error()})
		r.logger.Warn("failed to fetch albums", "error", err)
	}

	// Fetch library artists
	r.writePlain("👨‍🎤 Fetching artists...\n")
	if resp, err := r.api.Get(ctx, "/api/library/artists"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.Artists = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/library/artists", "error": err.Error()})
		r.logger.Warn("failed to fetch artists", "error", err)
	}

	// Fetch liked songs
	r.writePlain("❤️  Fetching liked songs...\n")
	if resp, err := r.api.Get(ctx, "/api/library/liked-songs"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.LikedSongs = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/library/liked-songs", "error": err.Error()})
		r.logger.Warn("failed to fetch liked songs", "error", err)
	}

	// Fetch history
	r.writePlain("📜 Fetching history...\n")
	if resp, err := r.api.Get(ctx, "/api/library/history"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.History = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/library/history", "error": err.Error()})
		r.logger.Warn("failed to fetch history", "error", err)
	}

	// Fetch uploaded songs
	r.writePlain("☁️  Fetching uploaded songs...\n")
	if resp, err := r.api.Get(ctx, "/api/uploads/songs"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.UploadedSongs = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/uploads/songs", "error": err.Error()})
		r.logger.Warn("failed to fetch uploaded songs", "error", err)
	}

	// Fetch uploaded albums
	r.writePlain("☁️  Fetching uploaded albums...\n")
	if resp, err := r.api.Get(ctx, "/api/uploads/albums"); err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
		dump.UploadedAlbums = resp.JSONData
	} else {
		dump.Errors = append(dump.Errors, map[string]string{"endpoint": "/api/uploads/albums", "error": err.Error()})
		r.logger.Warn("failed to fetch uploaded albums", "error", err)
	}

	r.writePlain("\n✓ Dump complete\n\n")

	// Save to file if requested
	if save {
		saveFile := "api_dump.json"
		data, err := shared.MarshalJSON(dump, true)
		if err != nil {
			return fmt.Errorf("failed to marshal dump: %w", err)
		}
		if err := os.WriteFile(saveFile, data, 0644); err != nil {
			r.logger.Warn("failed to save dump", "error", err)
		} else {
			r.logger.Info("dump saved", "file", saveFile)
			r.writePlain("✓ Dump saved to %s\n\n", saveFile)
		}
	}

	// Output to console
	return r.writeJSON(dump, pretty)
}

// apiCommand handles direct (proxy) API calls (v0.4) and dump (v0.7)
func apiCommand(r *Runner) *cli.Command {
	return &cli.Command{
		Name:  "api",
		Usage: "Direct API calls to FastAPI proxy",
		Commands: []*cli.Command{
			{
				Name:  "get",
				Usage: "Direct GET to FastAPI proxy, prints raw JSON",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "path",
					},
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "json",
						Usage: "Output raw JSON",
						Value: true,
					},
				},
				Action: r.APIGet,
			},
			{
				Name:  "post",
				Usage: "Direct POST with JSON body",
				Arguments: []cli.Argument{
					&cli.StringArg{
						Name: "path",
					},
				},
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "data",
						Aliases:  []string{"d"},
						Usage:    "JSON body to send",
						Required: true,
					},
				},
				Action: r.APIPost,
			},
			{
				Name:  "dump",
				Usage: "Full proxy state dump (cached playlists, songs, etc)",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "pretty",
						Usage: "Pretty-print output",
						Value: true,
					},
					&cli.BoolFlag{
						Name:  "save",
						Usage: "Save dump to api_dump.json",
						Value: false,
					},
				},
				Action: r.APIDump,
			},
		},
	}
}
