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
				},
				Action: r.TransferDiff,
			},
		},
	}
}
