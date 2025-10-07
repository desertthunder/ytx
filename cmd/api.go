// TODO: encapsulate file saving in [shared]
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/desertthunder/ytx/internal/shared"
	"github.com/desertthunder/ytx/internal/tasks"
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

	progressCh := make(chan tasks.ProgressUpdate, 20)
	go func() {
		// TODO: use unicode symbols where possible
		symbols := map[tasks.Phase]string{
			tasks.FetchHealth:    "📊",
			tasks.FetchPlaylists: "📝",
			tasks.FetchSongs:     "🎵",
			tasks.FetchAlbums:    "💿",
			tasks.FetchArtists:   "👨‍🎤",
			tasks.FetchLiked:     "❤️ ",
			tasks.FetchHistory:   "📜",
			tasks.FetchUploads:   "☁️ ",
		}
		for update := range progressCh {
			emoji := symbols[update.Phase]
			if emoji == "" {
				emoji = "📥"
			}
			r.writePlain("%s %s\n", emoji, update.Message)
		}
	}()

	result, err := r.engine.Dump(ctx, progressCh)
	close(progressCh)

	if err != nil {
		return err
	}

	for _, endpointErr := range result.Errors {
		r.logger.Warn("failed to fetch endpoint", "endpoint", endpointErr.Endpoint, "error", endpointErr.Error)
	}

	r.writePlain("\n✓ Dump complete\n\n")

	dump := tasks.DumpData{
		Health:         result.Health,
		Playlists:      result.Playlists,
		Songs:          result.Songs,
		Albums:         result.Albums,
		Artists:        result.Artists,
		LikedSongs:     result.LikedSongs,
		History:        result.History,
		UploadedSongs:  result.UploadedSongs,
		UploadedAlbums: result.UploadedAlbums,
		Errors:         []any{},
	}

	for _, endpointErr := range result.Errors {
		dump.Errors = append(dump.Errors, map[string]string{
			"endpoint": endpointErr.Endpoint,
			"error":    endpointErr.Error.Error(),
		})
	}

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

	return r.writeJSON(dump, pretty)
}

// apiCommand handles direct (proxy) API calls
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
