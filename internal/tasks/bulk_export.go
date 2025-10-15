package tasks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/desertthunder/ytx/internal/formatter"
	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/shared"
	"golang.org/x/time/rate"
)

// BulkExportOpts contains configuration for bulk playlist exports.
type BulkExportOpts struct {
	Format        string                                               // Export format: json, csv, markdown, txt
	OutputDir     string                                               // Base output directory (default: spotify_export_{epoch})
	NumWorkers    int                                                  // Concurrent workers (default: 5)
	RateLimit     float64                                              // Requests per second (default: 5)
	GetCoverImage func(ctx context.Context, id string) (string, error) // Fetcher function
}

// BulkExport exports multiple playlists concurrently with rate limiting and progress tracking.
//
// This method implements a worker pool pattern to efficiently export multiple playlists.
// It respects API rate limits, handles partial failures gracefully, and generates a manifest file summarizing the export results.
func (e *PlaylistEngine) BulkExport(
	ctx context.Context,
	prog chan<- ProgressUpdate,
	srv services.Service,
	ids []string,
	opts BulkExportOpts,
) (*BulkExportResult, error) {
	if srv == nil {
		return nil, fmt.Errorf("%w: service not initialized", shared.ErrServiceUnavailable)
	}

	if opts.OutputDir == "" {
		opts.OutputDir = fmt.Sprintf("spotify_export_%d", time.Now().Unix())
	}
	if opts.NumWorkers <= 0 {
		opts.NumWorkers = 5
	}
	if opts.NumWorkers > 10 {
		opts.NumWorkers = 10
	}
	if opts.RateLimit <= 0 {
		opts.RateLimit = 5.0
	}

	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	result := &BulkExportResult{
		TotalPlaylists:  len(ids),
		OutputDirectory: opts.OutputDir,
		Results:         make([]PlaylistExportResult, 0, len(ids)),
	}

	limiter := rate.NewLimiter(rate.Limit(opts.RateLimit), 1)

	jobs := make(chan PlaylistExportJob, len(ids))
	results := make(chan PlaylistExportResult, len(ids))

	var wg sync.WaitGroup
	for i := 0; i < opts.NumWorkers; i++ {
		wg.Add(1)
		go e.exportWorker(ctx, &wg, jobs, results, opts)
	}

	go func() {
		e.sendProgress(prog, fetchingSourceUpdate(1, len(ids)))
		for i, playlistID := range ids {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			default:
			}

			if err := limiter.Wait(ctx); err != nil {
				close(jobs)
				return
			}

			export, err := srv.ExportPlaylist(ctx, playlistID)
			if err != nil {
				results <- PlaylistExportResult{
					PlaylistID:   playlistID,
					PlaylistName: fmt.Sprintf("Unknown (%s)", playlistID),
					Success:      false,
					Error:        fmt.Errorf("failed to fetch playlist: %w", err),
				}
				continue
			}

			jobs <- PlaylistExportJob{
				PlaylistID: playlistID,
				Export:     export,
			}

			e.sendProgress(prog, exportingPlaylistUpdate(i+1, len(ids), export.Playlist.Name))
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	completed := 0
	for res := range results {
		completed++
		result.Results = append(result.Results, res)

		if res.Success {
			result.SuccessfulExports++
			e.sendProgress(prog, exportCompletedUpdate(
				completed,
				len(ids),
				res.PlaylistName,
				len(res.Files),
			))
		} else {
			result.FailedExports++
			e.sendProgress(prog, exportFailedUpdate(
				completed,
				len(ids),
				res.PlaylistName,
				res.Error,
			))
		}
	}

	manifestPath := filepath.Join(opts.OutputDir, "export_manifest.json")
	if err := formatter.WriteBulkExportManifest(result, opts.Format, manifestPath); err != nil {
		return result, fmt.Errorf("export completed but failed to write manifest: %w", err)
	}
	result.ManifestPath = manifestPath
	return result, nil
}

// exportWorker is a worker goroutine that exports playlists from the jobs channel.
func (e *PlaylistEngine) exportWorker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs <-chan PlaylistExportJob,
	results chan<- PlaylistExportResult,
	opts BulkExportOpts,
) {
	defer wg.Done()

	for job := range jobs {
		select {
		case <-ctx.Done():
			return
		default:
		}

		res := e.exportSinglePlaylist(ctx, job, opts)
		results <- res
	}
}

// exportSinglePlaylist exports a single playlist to the appropriate format.
func (e *PlaylistEngine) exportSinglePlaylist(
	ctx context.Context,
	j PlaylistExportJob,
	opts BulkExportOpts,
) PlaylistExportResult {
	result := PlaylistExportResult{
		PlaylistID:   j.PlaylistID,
		PlaylistName: j.Export.Playlist.Name,
		Success:      false,
		Files:        []string{},
	}

	switch opts.Format {
	case "csv":
		baseFilepath := filepath.Join(opts.OutputDir, j.Export.Playlist.ID)
		csvRes, err := formatter.WriteCSVExport(j.Export, baseFilepath)
		if err != nil {
			result.Error = fmt.Errorf("CSV export failed: %w", err)
			return result
		}
		result.Files = []string{csvRes.TracksFile, csvRes.MetadataFile}
		result.Success = true

	case "markdown":
		outputDir := filepath.Join(opts.OutputDir, j.Export.Playlist.ID)

		var imageURL string
		if opts.GetCoverImage != nil {
			if url, err := opts.GetCoverImage(ctx, j.PlaylistID); err == nil {
				imageURL = url
			}
		}

		mdRes, err := formatter.WriteMarkdownExport(j.Export, outputDir, imageURL)
		if err != nil {
			result.Error = fmt.Errorf("markdown export failed: %w", err)
			return result
		}
		result.Files = mdRes.Files
		result.Success = true

	case "txt":
		txtPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s_tracks.txt", j.Export.Playlist.ID))
		filepath, err := formatter.WriteTextExport(j.Export, txtPath)
		if err != nil {
			result.Error = fmt.Errorf("text export failed: %w", err)
			return result
		}
		result.Files = []string{filepath}
		result.Success = true
	case "json":
		fallthrough
	default:
		jsonPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s.json", j.Export.Playlist.ID))
		data, err := shared.MarshalJSON(j.Export, true)
		if err != nil {
			result.Error = fmt.Errorf("JSON marshal failed: %w", err)
			return result
		}
		if err := os.WriteFile(jsonPath, data, 0644); err != nil {
			result.Error = fmt.Errorf("JSON write failed: %w", err)
			return result
		}
		result.Files = []string{jsonPath}
		result.Success = true
	}
	return result
}
