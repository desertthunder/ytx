package main

import (
	"context"

	"github.com/desertthunder/song-migrations/internal/shared"
	"github.com/urfave/cli/v3"
)

// TransferRun runs a full Spotify → YouTube Music sync.
func (r *Runner) TransferRun(ctx context.Context, cmd *cli.Command) error {
	r.logger.Info("transfer run requested")
	return shared.ErrNotImplemented
}

// TransferDiff compares and show missing tracks between two playlists.
func (r *Runner) TransferDiff(ctx context.Context, cmd *cli.Command) error {
	r.logger.Info("transfer diff requested")
	return shared.ErrNotImplemented
}
