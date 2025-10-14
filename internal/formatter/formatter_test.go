package formatter

import (
	"strings"
	"testing"

	"github.com/desertthunder/ytx/internal/models"
	th "github.com/desertthunder/ytx/internal/testing"
)

func TestExporters(t *testing.T) {
	t.Run("ExportToCSV", func(t *testing.T) {
		export := &models.PlaylistExport{
			Playlist: models.Playlist{
				ID:          "test123",
				Name:        "Test Playlist",
				Description: "A test playlist",
				TrackCount:  2,
				Public:      true,
			},
			Tracks: []models.Track{
				{
					ID:       "track1",
					Title:    "Song One",
					Artist:   "Artist One",
					Album:    "Album One",
					Duration: 180,
					ISRC:     "USRC12345678",
				},
				{
					ID:       "track2",
					Title:    "Song Two",
					Artist:   "Artist Two",
					Album:    "Album Two",
					Duration: 240,
					ISRC:     "USRC87654321",
				},
			},
		}

		data, err := ExportToCSV(export)
		if err != nil {
			t.Fatalf("ExportToCSV failed: %v", err)
		}

		output := string(data)

		if !strings.Contains(output, "ID,Title,Artist,Album,Duration,ISRC") {
			t.Errorf("CSV missing headers, got: %s", output)
		}

		if !strings.Contains(output, "track1") {
			t.Errorf("CSV missing track1 ID")
		}
		if !strings.Contains(output, "Song One") {
			t.Errorf("CSV missing track1 title")
		}
		if !strings.Contains(output, "Artist One") {
			t.Errorf("CSV missing track1 artist")
		}
		if !strings.Contains(output, "USRC12345678") {
			t.Errorf("CSV missing track1 ISRC")
		}
	})

	t.Run("ExportToMarkdown", func(t *testing.T) {
		export := &models.PlaylistExport{
			Playlist: models.Playlist{
				ID:          "test123",
				Name:        "Test Playlist",
				Description: "A test playlist",
				TrackCount:  2,
				Public:      true,
			},
			Tracks: []models.Track{
				{
					ID:       "track1",
					Title:    "Song One",
					Artist:   "Artist One",
					Album:    "Album One",
					Duration: 180,
					ISRC:     "USRC12345678",
				},
				{
					ID:       "track2",
					Title:    "Song Two",
					Artist:   "Artist Two",
					Album:    "",
					Duration: 240,
					ISRC:     "USRC87654321",
				},
			},
		}

		t.Run("without cover image", func(t *testing.T) {
			data, err := ExportToMarkdown(export, "")
			if err != nil {
				t.Fatalf("ExportToMarkdown failed: %v", err)
			}

			output := string(data)

			if !strings.Contains(output, "# Test Playlist") {
				t.Errorf("Markdown missing title")
			}

			if !strings.Contains(output, "**Description**: A test playlist") {
				t.Errorf("Markdown missing description")
			}
			if !strings.Contains(output, "**Tracks**: 2") {
				t.Errorf("Markdown missing track count")
			}
			if !strings.Contains(output, "**Visibility**: Public") {
				t.Errorf("Markdown missing visibility")
			}

			if !strings.Contains(output, "## Tracks") {
				t.Errorf("Markdown missing tracks section")
			}
			if !strings.Contains(output, "1. Artist One - Song One (Album One) [3:00]") {
				t.Errorf("Markdown missing track1, got: %s", output)
			}
			if !strings.Contains(output, "2. Artist Two - Song Two [4:00]") {
				t.Errorf("Markdown missing track2 (no album)")
			}
		})

		t.Run("with cover image", func(t *testing.T) {
			data, err := ExportToMarkdown(export, "test_cover.jpg")
			if err != nil {
				t.Fatalf("ExportToMarkdown failed: %v", err)
			}

			output := string(data)

			if !strings.Contains(output, "![Cover](test_cover.jpg)") {
				t.Errorf("Markdown missing cover image reference")
			}
		})
	})

	t.Run("ExportToText", func(t *testing.T) {
		export := &models.PlaylistExport{
			Playlist: models.Playlist{
				ID:          "test123",
				Name:        "Test Playlist",
				Description: "A test playlist",
				TrackCount:  2,
				Public:      false,
			},
			Tracks: []models.Track{
				{
					ID:       "track1",
					Title:    "Song One",
					Artist:   "Artist One",
					Album:    "Album One",
					Duration: 180,
				},
				{
					ID:       "track2",
					Title:    "Song Two",
					Artist:   "Artist Two",
					Album:    "Album Two",
					Duration: 240,
				},
			},
		}

		data, err := ExportToText(export)
		if err != nil {
			t.Fatalf("ExportToText failed: %v", err)
		}

		output := string(data)

		if !strings.Contains(output, "Playlist: Test Playlist") {
			t.Errorf("Text missing playlist name")
		}
		if !strings.Contains(output, "Description: A test playlist") {
			t.Errorf("Text missing description")
		}
		if !strings.Contains(output, "Tracks: 2") {
			t.Errorf("Text missing track count")
		}

		if !strings.Contains(output, "1. Artist One - Song One") {
			t.Errorf("Text missing track1")
		}
		if !strings.Contains(output, "2. Artist Two - Song Two") {
			t.Errorf("Text missing track2")
		}
	})

	t.Run("ToMetadataJSON", func(t *testing.T) {
		playlist := models.Playlist{
			ID:          "test123",
			Name:        "Test Playlist",
			Description: "A test playlist",
			TrackCount:  10,
			Public:      true,
		}

		data, err := ToMetadataJSON(playlist)
		if err != nil {
			t.Fatalf("GenerateMetadataJSON failed: %v", err)
		}

		output := string(data)

		if !strings.Contains(output, `"ID":"test123"`) && !strings.Contains(output, `"ID": "test123"`) {
			t.Errorf("JSON missing ID field")
		}
		if !strings.Contains(output, `"Name":"Test Playlist"`) && !strings.Contains(output, `"Name": "Test Playlist"`) {
			t.Errorf("JSON missing Name field")
		}
	})
}

func TestDownloadImage(t *testing.T) {
	t.Run("EmptyURL", func(t *testing.T) {
		_, err := DownloadImage("")
		if err == nil {
			t.Error("DownloadImage with empty URL should return error")
		}
	})
}

func TestWriters(t *testing.T) {
	t.Run("WriteCSVExport", func(t *testing.T) {
		export := &models.PlaylistExport{
			Playlist: models.Playlist{
				ID:          "test123",
				Name:        "Test Playlist",
				Description: "A test playlist",
				TrackCount:  2,
				Public:      true,
			},
			Tracks: []models.Track{
				{
					ID:       "track1",
					Title:    "Song One",
					Artist:   "Artist One",
					Album:    "Album One",
					Duration: 180,
					ISRC:     "USRC12345678",
				},
				{
					ID:       "track2",
					Title:    "Song Two",
					Artist:   "Artist Two",
					Album:    "Album Two",
					Duration: 240,
					ISRC:     "USRC87654321",
				},
			},
		}

		t.Run("WithDefaultPath", func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir := th.MustGetwd(t)
			th.MustChdir(t, tempDir)
			defer th.MustChdir(t, originalDir)

			result, err := WriteCSVExport(export, "")
			if err != nil {
				t.Fatalf("WriteCSVExport failed: %v", err)
			}

			if result.TracksFile != "test123_tracks.csv" {
				t.Errorf("Expected tracks file 'test123_tracks.csv', got '%s'", result.TracksFile)
			}
			if result.MetadataFile != "test123_metadata.json" {
				t.Errorf("Expected metadata file 'test123_metadata.json', got '%s'", result.MetadataFile)
			}

			th.AssertFileExists(t, result.TracksFile)
			th.AssertFileExists(t, result.MetadataFile)

			csvContent := th.MustReadFile(t, result.TracksFile)
			if !strings.Contains(csvContent, "ID,Title,Artist,Album,Duration,ISRC") {
				t.Errorf("CSV missing headers")
			}
			if !strings.Contains(csvContent, "track1") || !strings.Contains(csvContent, "Song One") {
				t.Errorf("CSV missing track data")
			}

			metadataContent := th.MustReadFile(t, result.MetadataFile)
			if !strings.Contains(metadataContent, "test123") || !strings.Contains(metadataContent, "Test Playlist") {
				t.Errorf("Metadata JSON missing expected fields")
			}
		})

		t.Run("WithCustomPath", func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir := th.MustGetwd(t)
			th.MustChdir(t, tempDir)
			defer th.MustChdir(t, originalDir)

			result, err := WriteCSVExport(export, "custom_export")
			if err != nil {
				t.Fatalf("WriteCSVExport failed: %v", err)
			}

			if result.TracksFile != "custom_export_tracks.csv" {
				t.Errorf("Expected 'custom_export_tracks.csv', got '%s'", result.TracksFile)
			}
			if result.MetadataFile != "custom_export_metadata.json" {
				t.Errorf("Expected 'custom_export_metadata.json', got '%s'", result.MetadataFile)
			}

			th.AssertFileExists(t, result.TracksFile)
			th.AssertFileExists(t, result.MetadataFile)
		})
	})

	t.Run("WriteMarkdownExport", func(t *testing.T) {
		export := &models.PlaylistExport{
			Playlist: models.Playlist{
				ID:          "test123",
				Name:        "Test Playlist",
				Description: "A test playlist",
				TrackCount:  2,
				Public:      true,
			},
			Tracks: []models.Track{
				{
					ID:       "track1",
					Title:    "Song One",
					Artist:   "Artist One",
					Album:    "Album One",
					Duration: 180,
				},
				{
					ID:       "track2",
					Title:    "Song Two",
					Artist:   "Artist Two",
					Duration: 240,
				},
			},
		}

		t.Run("WithDefaultDirectory", func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir := th.MustGetwd(t)
			th.MustChdir(t, tempDir)
			defer th.MustChdir(t, originalDir)

			result, err := WriteMarkdownExport(export, "", "")
			if err != nil {
				t.Fatalf("WriteMarkdownExport failed: %v", err)
			}

			if result.Directory != "test123" {
				t.Errorf("Expected directory 'test123', got '%s'", result.Directory)
			}
			th.AssertDirExists(t, result.Directory)

			readmePath := result.Directory + "/README.md"
			th.AssertFileExists(t, readmePath)

			content := th.MustReadFile(t, readmePath)
			if !strings.Contains(content, "# Test Playlist") {
				t.Errorf("Markdown missing title")
			}
			if !strings.Contains(content, "1. Artist One - Song One (Album One)") {
				t.Errorf("Markdown missing track listing")
			}

			if result.CoverImage != "" {
				t.Errorf("Expected no cover image, got '%s'", result.CoverImage)
			}
		})

		t.Run("WithCustomDirectory", func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir := th.MustGetwd(t)
			th.MustChdir(t, tempDir)
			defer th.MustChdir(t, originalDir)

			result, err := WriteMarkdownExport(export, "custom_playlist", "")
			if err != nil {
				t.Fatalf("WriteMarkdownExport failed: %v", err)
			}

			if result.Directory != "custom_playlist" {
				t.Errorf("Expected directory 'custom_playlist', got '%s'", result.Directory)
			}
			th.AssertDirExists(t, result.Directory)
			th.AssertFileExists(t, result.Directory+"/README.md")
		})
	})

	t.Run("TestWriteTextExport", func(t *testing.T) {
		export := &models.PlaylistExport{
			Playlist: models.Playlist{
				ID:          "test123",
				Name:        "Test Playlist",
				Description: "A test playlist",
				TrackCount:  2,
				Public:      false,
			},
			Tracks: []models.Track{
				{
					ID:       "track1",
					Title:    "Song One",
					Artist:   "Artist One",
					Duration: 180,
				},
				{
					ID:       "track2",
					Title:    "Song Two",
					Artist:   "Artist Two",
					Duration: 240,
				},
			},
		}

		t.Run("WithDefaultPath", func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir := th.MustGetwd(t)
			th.MustChdir(t, tempDir)
			defer th.MustChdir(t, originalDir)

			filepath, err := WriteTextExport(export, "")
			if err != nil {
				t.Fatalf("WriteTextExport failed: %v", err)
			}

			if filepath != "test123_tracks.txt" {
				t.Errorf("Expected 'test123_tracks.txt', got '%s'", filepath)
			}

			th.AssertFileExists(t, filepath)

			content := th.MustReadFile(t, filepath)
			if !strings.Contains(content, "Playlist: Test Playlist") {
				t.Errorf("Text missing playlist name")
			}
			if !strings.Contains(content, "1. Artist One - Song One") {
				t.Errorf("Text missing track listing")
			}
		})

		t.Run("WithCustomPath", func(t *testing.T) {
			tempDir := t.TempDir()
			originalDir := th.MustGetwd(t)
			th.MustChdir(t, tempDir)
			defer th.MustChdir(t, originalDir)

			filepath, err := WriteTextExport(export, "my_playlist.txt")
			if err != nil {
				t.Fatalf("WriteTextExport failed: %v", err)
			}

			if filepath != "my_playlist.txt" {
				t.Errorf("Expected 'my_playlist.txt', got '%s'", filepath)
			}

			th.AssertFileExists(t, filepath)
		})
	})
}
