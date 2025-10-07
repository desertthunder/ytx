package main

import (
	"context"
	"encoding/json"
	"fmt"

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
