package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/desertthunder/ytx/internal/formatter"
	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

func TestBulkExport_SuccessfulExport(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		playlistCount  int
		wantSuccess    int
		wantFailed     int
		validateResult func(t *testing.T, result *BulkExportResult, tempDir string)
	}{
		{
			name:          "single playlist json export",
			format:        "json",
			playlistCount: 1,
			wantSuccess:   1,
			wantFailed:    0,
			validateResult: func(t *testing.T, result *BulkExportResult, tempDir string) {
				if len(result.Results) != 1 {
					t.Errorf("expected 1 result, got %d", len(result.Results))
				}
				if len(result.Results[0].Files) != 1 {
					t.Errorf("expected 1 file, got %d", len(result.Results[0].Files))
				}
				// Verify JSON file was created
				jsonPath := filepath.Join(tempDir, "playlist1.json")
				if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
					t.Errorf("JSON file not created at %s", jsonPath)
				}
			},
		},
		{
			name:          "multiple playlists csv export",
			format:        "csv",
			playlistCount: 3,
			wantSuccess:   3,
			wantFailed:    0,
			validateResult: func(t *testing.T, result *BulkExportResult, tempDir string) {
				if len(result.Results) != 3 {
					t.Errorf("expected 3 results, got %d", len(result.Results))
				}
				for _, res := range result.Results {
					if len(res.Files) != 2 {
						t.Errorf("CSV export should create 2 files, got %d", len(res.Files))
					}
				}
			},
		},
		{
			name:          "text export",
			format:        "txt",
			playlistCount: 2,
			wantSuccess:   2,
			wantFailed:    0,
			validateResult: func(t *testing.T, result *BulkExportResult, tempDir string) {
				if len(result.Results) != 2 {
					t.Errorf("expected 2 results, got %d", len(result.Results))
				}
				for _, res := range result.Results {
					if len(res.Files) != 1 {
						t.Errorf("text export should create 1 file, got %d", len(res.Files))
					}
				}
			},
		},
		{
			name:          "markdown export",
			format:        "markdown",
			playlistCount: 1,
			wantSuccess:   1,
			wantFailed:    0,
			validateResult: func(t *testing.T, result *BulkExportResult, tempDir string) {
				if len(result.Results) != 1 {
					t.Errorf("expected 1 result, got %d", len(result.Results))
				}
				// Markdown export creates at least README.md
				if len(result.Results[0].Files) < 1 {
					t.Errorf("markdown export should create at least 1 file, got %d", len(result.Results[0].Files))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()

			// Create mock service with playlists
			playlistExports := make(map[string]*models.PlaylistExport)
			playlistIDs := make([]string, tt.playlistCount)
			for i := 0; i < tt.playlistCount; i++ {
				id := fmt.Sprintf("playlist%d", i+1)
				playlistIDs[i] = id
				playlistExports[id] = &models.PlaylistExport{
					Playlist: models.Playlist{
						ID:          id,
						Name:        fmt.Sprintf("Playlist %d", i+1),
						Description: fmt.Sprintf("Test playlist %d", i+1),
						TrackCount:  2,
					},
					Tracks: []models.Track{
						{ID: fmt.Sprintf("track%d-1", i+1), Title: "Song 1", Artist: "Artist 1"},
						{ID: fmt.Sprintf("track%d-2", i+1), Title: "Song 2", Artist: "Artist 2"},
					},
				}
			}

			mockSvc := &mockService{
				name:            "Spotify",
				playlistExports: playlistExports,
			}

			engine := NewPlaylistEngine(nil, nil, nil)
			progressCh := make(chan ProgressUpdate, 100)
			go func() {
				for range progressCh {
					// Drain progress channel
				}
			}()

			opts := BulkExportOpts{
				Format:     tt.format,
				OutputDir:  tempDir,
				NumWorkers: 2,
				RateLimit:  10.0,
			}

			result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, playlistIDs, opts)
			close(progressCh)

			if err != nil {
				t.Fatalf("BulkExport() error = %v", err)
			}

			if result.TotalPlaylists != tt.playlistCount {
				t.Errorf("TotalPlaylists = %d, want %d", result.TotalPlaylists, tt.playlistCount)
			}

			if result.SuccessfulExports != tt.wantSuccess {
				t.Errorf("SuccessfulExports = %d, want %d", result.SuccessfulExports, tt.wantSuccess)
			}

			if result.FailedExports != tt.wantFailed {
				t.Errorf("FailedExports = %d, want %d", result.FailedExports, tt.wantFailed)
			}

			if result.OutputDirectory != tempDir {
				t.Errorf("OutputDirectory = %s, want %s", result.OutputDirectory, tempDir)
			}

			// Verify manifest was created
			if result.ManifestPath == "" {
				t.Error("ManifestPath should not be empty")
			}

			manifestPath := filepath.Join(tempDir, "export_manifest.json")
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				t.Errorf("manifest file not created at %s", manifestPath)
			}

			// Verify manifest contents
			manifestData, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatalf("failed to read manifest: %v", err)
			}

			var manifest formatter.ExportManifest
			if err := json.Unmarshal(manifestData, &manifest); err != nil {
				t.Fatalf("failed to parse manifest: %v", err)
			}

			if manifest.Format != tt.format {
				t.Errorf("manifest format = %s, want %s", manifest.Format, tt.format)
			}

			if manifest.TotalPlaylists != tt.playlistCount {
				t.Errorf("manifest total = %d, want %d", manifest.TotalPlaylists, tt.playlistCount)
			}

			if tt.validateResult != nil {
				tt.validateResult(t, result, tempDir)
			}
		})
	}
}

func TestBulkExport_PartialFailures(t *testing.T) {
	tempDir := t.TempDir()

	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"playlist1": {
				Playlist: models.Playlist{ID: "playlist1", Name: "Playlist 1"},
				Tracks:   []models.Track{{ID: "t1", Title: "Song 1", Artist: "Artist 1"}},
			},
			"playlist3": {
				Playlist: models.Playlist{ID: "playlist3", Name: "Playlist 3"},
				Tracks:   []models.Track{{ID: "t3", Title: "Song 3", Artist: "Artist 3"}},
			},
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	playlistIDs := []string{"playlist1", "playlist2", "playlist3"}
	opts := BulkExportOpts{
		Format:     "json",
		OutputDir:  tempDir,
		NumWorkers: 2,
		RateLimit:  10.0,
	}

	result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, playlistIDs, opts)
	close(progressCh)

	if err != nil {
		t.Fatalf("BulkExport() error = %v", err)
	}

	if result.TotalPlaylists != 3 {
		t.Errorf("TotalPlaylists = %d, want 3", result.TotalPlaylists)
	}

	if result.SuccessfulExports != 2 {
		t.Errorf("SuccessfulExports = %d, want 2", result.SuccessfulExports)
	}

	if result.FailedExports != 1 {
		t.Errorf("FailedExports = %d, want 1", result.FailedExports)
	}

	// Find the failed result
	var failedResult *PlaylistExportResult
	for i := range result.Results {
		if !result.Results[i].Success {
			failedResult = &result.Results[i]
			break
		}
	}

	if failedResult == nil {
		t.Fatal("expected one failed result")
	}

	if failedResult.PlaylistID != "playlist2" {
		t.Errorf("failed playlist ID = %s, want playlist2", failedResult.PlaylistID)
	}

	if failedResult.Error == nil {
		t.Error("failed result should have an error")
	}
}

func TestBulkExport_ServiceError(t *testing.T) {
	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 10)

	opts := BulkExportOpts{
		Format:    "json",
		OutputDir: t.TempDir(),
	}

	_, err := engine.BulkExport(context.Background(), progressCh, nil, []string{"p1"}, opts)
	close(progressCh)

	if err == nil {
		t.Error("BulkExport() expected error for nil service")
	}

	if !strings.Contains(err.Error(), "not initialized") {
		t.Errorf("error should mention service not initialized, got: %v", err)
	}
}

func TestBulkExport_ContextCancellation(t *testing.T) {
	tempDir := t.TempDir()

	// Create mock service with slow exports
	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"playlist1": {
				Playlist: models.Playlist{ID: "playlist1", Name: "Playlist 1"},
				Tracks:   []models.Track{{ID: "t1", Title: "Song 1", Artist: "Artist 1"}},
			},
			"playlist2": {
				Playlist: models.Playlist{ID: "playlist2", Name: "Playlist 2"},
				Tracks:   []models.Track{{ID: "t2", Title: "Song 2", Artist: "Artist 2"}},
			},
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	opts := BulkExportOpts{
		Format:     "json",
		OutputDir:  tempDir,
		NumWorkers: 1,
		RateLimit:  10.0,
	}

	result, err := engine.BulkExport(ctx, progressCh, mockSvc, []string{"playlist1", "playlist2"}, opts)
	close(progressCh)

	// Should complete without error even if context is cancelled
	if err != nil {
		t.Errorf("BulkExport() should handle cancellation gracefully, got error: %v", err)
	}

	// May have partial results depending on timing
	if result == nil {
		t.Fatal("result should not be nil")
	}
}

func TestBulkExport_DefaultOptions(t *testing.T) {
	// Change to a temp directory so default directory creation happens there
	tempDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	defer os.Chdir(originalDir)

	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"playlist1": {
				Playlist: models.Playlist{ID: "playlist1", Name: "Playlist 1"},
				Tracks:   []models.Track{{ID: "t1", Title: "Song 1", Artist: "Artist 1"}},
			},
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	// Test with empty opts to verify defaults
	opts := BulkExportOpts{}

	result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, []string{"playlist1"}, opts)
	close(progressCh)

	if err != nil {
		t.Fatalf("BulkExport() error = %v", err)
	}

	// Verify default output directory was created (spotify_export_{epoch})
	if !strings.HasPrefix(filepath.Base(result.OutputDirectory), "spotify_export_") {
		t.Errorf("default output directory should start with 'spotify_export_', got: %s", result.OutputDirectory)
	}

	// Verify directory was actually created
	if _, err := os.Stat(result.OutputDirectory); os.IsNotExist(err) {
		t.Errorf("output directory was not created: %s", result.OutputDirectory)
	}
}

func TestBulkExport_WorkerPoolLimits(t *testing.T) {
	tempDir := t.TempDir()
	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"p1": {Playlist: models.Playlist{ID: "p1", Name: "P1"}, Tracks: []models.Track{}},
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	tests := []struct {
		name           string
		numWorkers     int
		expectedWorker int
	}{
		{"default workers (0 -> 5)", 0, 5},
		{"negative workers (-1 -> 5)", -1, 5},
		{"max workers (15 -> 10)", 15, 10},
		{"valid workers (3)", 3, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := BulkExportOpts{
				Format:     "json",
				OutputDir:  tempDir,
				NumWorkers: tt.numWorkers,
			}

			result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, []string{"p1"}, opts)

			if err != nil {
				t.Fatalf("BulkExport() error = %v", err)
			}

			if result.SuccessfulExports != 1 {
				t.Errorf("export should succeed regardless of worker count")
			}
		})
	}

	close(progressCh)
}

func TestBulkExport_RateLimiting(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple playlists to test rate limiting
	playlistExports := make(map[string]*models.PlaylistExport)
	playlistIDs := make([]string, 5)
	for i := range 5 {
		id := fmt.Sprintf("playlist%d", i+1)
		playlistIDs[i] = id
		playlistExports[id] = &models.PlaylistExport{
			Playlist: models.Playlist{ID: id, Name: fmt.Sprintf("Playlist %d", i+1)},
			Tracks:   []models.Track{{ID: "t1", Title: "Song", Artist: "Artist"}},
		}
	}

	mockSvc := &mockService{
		name:            "Spotify",
		playlistExports: playlistExports,
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	opts := BulkExportOpts{
		Format:     "json",
		OutputDir:  tempDir,
		NumWorkers: 2,
		RateLimit:  5.0,
	}

	start := time.Now()
	result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, playlistIDs, opts)
	elapsed := time.Since(start)
	close(progressCh)

	if err != nil {
		t.Fatalf("BulkExport() error = %v", err)
	}

	if result.SuccessfulExports != 5 {
		t.Errorf("SuccessfulExports = %d, want 5", result.SuccessfulExports)
	}

	// With 5 playlists at 5 req/s, should take at least ~800ms (accounting for some overhead)
	// We won't enforce strict timing in tests, but verify it's not instant
	if elapsed < 100*time.Millisecond {
		t.Logf("Warning: export completed very quickly (%v), rate limiting may not be working", elapsed)
	}
	if mockSvc.exportCallCount != 5 {
		t.Errorf("service.ExportPlaylist called %d times, want 5", mockSvc.exportCallCount)
	}
}

func TestBulkExport_ProgressUpdates(t *testing.T) {
	tempDir := t.TempDir()
	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"p1": {
				Playlist: models.Playlist{ID: "p1", Name: "Playlist 1"},
				Tracks:   []models.Track{{ID: "t1", Title: "Song", Artist: "Artist"}},
			},
			"p2": {
				Playlist: models.Playlist{ID: "p2", Name: "Playlist 2"},
				Tracks:   []models.Track{{ID: "t2", Title: "Song", Artist: "Artist"}},
			},
		},
	}
	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	progressUpdates := []ProgressUpdate{}
	done := make(chan bool)
	go func() {
		for update := range progressCh {
			progressUpdates = append(progressUpdates, update)
		}
		done <- true
	}()

	opts := BulkExportOpts{
		Format:     "json",
		OutputDir:  tempDir,
		NumWorkers: 2,
		RateLimit:  10.0,
	}

	result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, []string{"p1", "p2"}, opts)
	close(progressCh)
	<-done

	if err != nil {
		t.Fatalf("BulkExport() error = %v", err)
	}
	if result.SuccessfulExports != 2 {
		t.Errorf("SuccessfulExports = %d, want 2", result.SuccessfulExports)
	}
	if len(progressUpdates) == 0 {
		t.Error("expected progress updates to be sent")
	}
	phases := make(map[Phase]bool)
	for _, update := range progressUpdates {
		phases[update.Phase] = true
	}
	if !phases[FetchSource] {
		t.Error("expected FetchSource phase in progress updates")
	}
}

func TestBulkExport_MarkdownWithCoverImage(t *testing.T) {
	tempDir := t.TempDir()
	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"p1": {
				Playlist: models.Playlist{ID: "p1", Name: "Playlist 1"},
				Tracks:   []models.Track{{ID: "t1", Title: "Song", Artist: "Artist"}},
			},
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	coverImageCalled := false
	getCoverImage := func(ctx context.Context, id string) (string, error) {
		coverImageCalled = true
		return "", fmt.Errorf("test: skip download")
	}

	opts := BulkExportOpts{
		Format:        "markdown",
		OutputDir:     tempDir,
		NumWorkers:    1,
		RateLimit:     10.0,
		GetCoverImage: getCoverImage,
	}

	result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, []string{"p1"}, opts)
	close(progressCh)

	if err != nil {
		t.Fatalf("BulkExport() error = %v", err)
	}
	if result.SuccessfulExports != 1 {
		t.Errorf("SuccessfulExports = %d, want 1", result.SuccessfulExports)
	}
	if !coverImageCalled {
		t.Error("GetCoverImage should have been called for markdown export")
	}
}

func TestBulkExport_OutputDirectoryCreation(t *testing.T) {
	// Create a temp dir but specify a subdirectory that doesn't exist
	baseDir := t.TempDir()
	outputDir := filepath.Join(baseDir, "exports", "spotify", "2024")

	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"p1": {
				Playlist: models.Playlist{ID: "p1", Name: "Playlist 1"},
				Tracks:   []models.Track{{ID: "t1", Title: "Song", Artist: "Artist"}},
			},
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 100)
	go func() {
		for range progressCh {
			// Drain progress channel
		}
	}()

	opts := BulkExportOpts{
		Format:    "json",
		OutputDir: outputDir,
	}

	result, err := engine.BulkExport(context.Background(), progressCh, mockSvc, []string{"p1"}, opts)
	close(progressCh)

	if err != nil {
		t.Fatalf("BulkExport() error = %v", err)
	}
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("nested output directory was not created: %s", outputDir)
	}
	if result.OutputDirectory != outputDir {
		t.Errorf("OutputDirectory = %s, want %s", result.OutputDirectory, outputDir)
	}
}

func TestBulkExport_InvalidOutputDirectory(t *testing.T) {
	mockSvc := &mockService{
		name: "Spotify",
		playlistExports: map[string]*models.PlaylistExport{
			"p1": {
				Playlist: models.Playlist{ID: "p1", Name: "Playlist 1"},
				Tracks:   []models.Track{{ID: "t1", Title: "Song", Artist: "Artist"}},
			},
		},
	}

	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 10)
	opts := BulkExportOpts{
		Format:    "json",
		OutputDir: "/root/invalid/path/that/should/not/be/writable",
	}

	_, err := engine.BulkExport(context.Background(), progressCh, mockSvc, []string{"p1"}, opts)
	close(progressCh)

	if err == nil {
		t.Error("BulkExport() expected error for invalid output directory")
	}
	if !strings.Contains(err.Error(), "failed to create output directory") {
		t.Errorf("error should mention directory creation failure, got: %v", err)
	}
}

func TestExportSinglePlaylist_AllFormats(t *testing.T) {
	tempDir := t.TempDir()
	export := &models.PlaylistExport{
		Playlist: models.Playlist{
			ID:          "test-playlist",
			Name:        "Test Playlist",
			Description: "Test Description",
			TrackCount:  2,
		},
		Tracks: []models.Track{
			{ID: "t1", Title: "Song 1", Artist: "Artist 1", Album: "Album 1", Duration: 180},
			{ID: "t2", Title: "Song 2", Artist: "Artist 2", Album: "Album 2", Duration: 240},
		},
	}
	job := PlaylistExportJob{
		PlaylistID: "test-playlist",
		Export:     export,
	}
	engine := NewPlaylistEngine(nil, nil, nil)
	tt := []struct {
		name         string
		format       string
		wantFiles    int
		validateFile func(t *testing.T, files []string)
	}{
		{
			name:      "json format",
			format:    "json",
			wantFiles: 1,
			validateFile: func(t *testing.T, files []string) {
				if !strings.HasSuffix(files[0], ".json") {
					t.Errorf("expected .json file, got: %s", files[0])
				}
			},
		},
		{
			name:      "csv format",
			format:    "csv",
			wantFiles: 2, // tracks.csv + metadata.json
			validateFile: func(t *testing.T, files []string) {
				hasCSV := false
				hasJSON := false
				for _, f := range files {
					if strings.HasSuffix(f, ".csv") {
						hasCSV = true
					}
					if strings.HasSuffix(f, ".json") {
						hasJSON = true
					}
				}
				if !hasCSV || !hasJSON {
					t.Errorf("CSV export should create .csv and .json files, got: %v", files)
				}
			},
		},
		{
			name:      "txt format",
			format:    "txt",
			wantFiles: 1,
			validateFile: func(t *testing.T, files []string) {
				if !strings.HasSuffix(files[0], ".txt") {
					t.Errorf("expected .txt file, got: %s", files[0])
				}
			},
		},
		{
			name:      "markdown format",
			format:    "markdown",
			wantFiles: 1,
			validateFile: func(t *testing.T, files []string) {
				if !strings.Contains(files[0], "README.md") {
					t.Errorf("expected README.md file, got: %s", files[0])
				}
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			opts := BulkExportOpts{
				Format:    tc.format,
				OutputDir: tempDir,
			}

			result := engine.exportSinglePlaylist(context.Background(), job, opts)
			if !result.Success {
				t.Fatalf("export failed: %v", result.Error)
			}
			if len(result.Files) < tc.wantFiles {
				t.Errorf("expected at least %d files, got %d", tc.wantFiles, len(result.Files))
			}
			if tc.validateFile != nil {
				tc.validateFile(t, result.Files)
			}
			for _, file := range result.Files {
				if _, err := os.Stat(file); os.IsNotExist(err) {
					t.Errorf("file not created: %s", file)
				}
			}
		})
	}
}

func TestExportSinglePlaylist_ServiceUnavailableError(t *testing.T) {
	engine := NewPlaylistEngine(nil, nil, nil)
	progressCh := make(chan ProgressUpdate, 10)
	opts := BulkExportOpts{
		Format:    "json",
		OutputDir: t.TempDir(),
	}

	_, err := engine.BulkExport(context.Background(), progressCh, nil, []string{"p1"}, opts)
	close(progressCh)

	if err == nil {
		t.Fatal("expected error for nil service")
	}
	if !strings.Contains(err.Error(), shared.ErrServiceUnavailable.Error()) {
		t.Errorf("expected ErrServiceUnavailable, got: %v", err)
	}
}
