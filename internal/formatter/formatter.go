// package formatter provides functions to export playlist data to various formats (CSV, Markdown, plain text)
package formatter

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

// ExportToCSV converts a PlaylistExport to CSV format with columns: ID, Title, Artist, Album, Duration, ISRC
func ExportToCSV(export *models.PlaylistExport) ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)

	headers := []string{"ID", "Title", "Artist", "Album", "Duration", "ISRC"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}

	for _, track := range export.Tracks {
		record := []string{
			track.ID,
			track.Title,
			track.Artist,
			track.Album,
			strconv.Itoa(track.Duration),
			track.ISRC,
		}
		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write CSV record: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("CSV writer error: %w", err)
	}

	return buf.Bytes(), nil
}

// ExportToMarkdown converts a PlaylistExport to Markdown format with optional cover image
func ExportToMarkdown(export *models.PlaylistExport, imageFilename string) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("# %s\n\n", export.Playlist.Name))

	if imageFilename != "" {
		buf.WriteString(fmt.Sprintf("![Cover](%s)\n\n", imageFilename))
	}

	if export.Playlist.Description != "" {
		buf.WriteString(fmt.Sprintf("**Description**: %s\n\n", export.Playlist.Description))
	}

	buf.WriteString(fmt.Sprintf("**Tracks**: %d\n", len(export.Tracks)))
	buf.WriteString(fmt.Sprintf("**Visibility**: %s\n\n", shared.VisibilityString(export.Playlist.Public)))

	buf.WriteString("## Tracks\n\n")
	for i, track := range export.Tracks {
		duration := shared.FormatDuration(track.Duration)
		albumPart := ""
		if track.Album != "" {
			albumPart = fmt.Sprintf(" (%s)", track.Album)
		}
		buf.WriteString(fmt.Sprintf("%d. %s - %s%s [%s]\n", i+1, track.Artist, track.Title, albumPart, duration))
	}

	return buf.Bytes(), nil
}

// ExportToText converts a PlaylistExport to plain text format
func ExportToText(export *models.PlaylistExport) ([]byte, error) {
	var buf bytes.Buffer

	buf.WriteString(fmt.Sprintf("Playlist: %s\n", export.Playlist.Name))
	if export.Playlist.Description != "" {
		buf.WriteString(fmt.Sprintf("Description: %s\n", export.Playlist.Description))
	}
	buf.WriteString(fmt.Sprintf("Tracks: %d\n\n", len(export.Tracks)))

	for i, track := range export.Tracks {
		buf.WriteString(fmt.Sprintf("%d. %s - %s\n", i+1, track.Artist, track.Title))
	}

	return buf.Bytes(), nil
}

// DownloadImage downloads an image from the given URL and returns the raw bytes
func DownloadImage(url string) ([]byte, error) {
	if url == "" {
		return nil, fmt.Errorf("empty URL provided")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image: status %d", resp.StatusCode)
	}

	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	return imageData, nil
}

// ToMetadataJSON generates a JSON representation of playlist metadata (without tracks)
func ToMetadataJSON(playlist models.Playlist) ([]byte, error) {
	return shared.MarshalJSON(playlist, true)
}

// CSVExportResult contains the paths of files created by WriteCSVExport
type CSVExportResult struct {
	TracksFile   string
	MetadataFile string
}

// WriteCSVExport exports a playlist to CSV format with accompanying metadata JSON file.
//
// Defaults to playlist ID as the base filename & creates {base}_tracks.csv and {base}_metadata.json
func WriteCSVExport(export *models.PlaylistExport, baseFilepath string) (*CSVExportResult, error) {
	if baseFilepath == "" {
		baseFilepath = export.Playlist.ID
	}

	csvData, err := ExportToCSV(export)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CSV: %w", err)
	}

	tracksFile := baseFilepath + "_tracks.csv"
	if err := os.WriteFile(tracksFile, csvData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write CSV file: %w", err)
	}

	metadataJSON, err := ToMetadataJSON(export.Playlist)
	if err != nil {
		return nil, fmt.Errorf("failed to generate metadata JSON: %w", err)
	}

	metadataFile := baseFilepath + "_metadata.json"
	if err := os.WriteFile(metadataFile, metadataJSON, 0644); err != nil {
		return nil, fmt.Errorf("failed to write metadata file: %w", err)
	}

	return &CSVExportResult{
		TracksFile:   tracksFile,
		MetadataFile: metadataFile,
	}, nil
}

// MarkdownExportResult contains information about files created by WriteMarkdownExport
type MarkdownExportResult struct {
	Directory  string
	Files      []string
	CoverImage string
}

// WriteMarkdownExport exports a playlist to Markdown format in a dedicated directory.
//
// Directory name defaults to the playlist ID.
// The imageURL parameter is optional - if provided, attempts to download the cover image.
// Creates a directory structure: {dir}/README.md and optionally {dir}/cover.jpg
func WriteMarkdownExport(export *models.PlaylistExport, outputDir string, imageURL string) (*MarkdownExportResult, error) {
	if outputDir == "" {
		outputDir = export.Playlist.ID
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	result := &MarkdownExportResult{
		Directory: outputDir,
		Files:     []string{},
	}

	var coverImageFilename string
	if imageURL != "" {
		imageData, err := DownloadImage(imageURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download cover image: %v\n", err)
		} else {
			coverImageFilename = "cover.jpg"
			coverImagePath := fmt.Sprintf("%s/%s", outputDir, coverImageFilename)
			if err := os.WriteFile(coverImagePath, imageData, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to save cover image: %v\n", err)
				coverImageFilename = ""
			} else {
				result.CoverImage = coverImagePath
				result.Files = append(result.Files, coverImagePath)
			}
		}
	}

	mdData, err := ExportToMarkdown(export, coverImageFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Markdown: %w", err)
	}

	mdFile := fmt.Sprintf("%s/README.md", outputDir)
	if err := os.WriteFile(mdFile, mdData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write Markdown file: %w", err)
	}

	result.Files = append(result.Files, mdFile)

	return result, nil
}

// WriteTextExport exports a playlist to plain text format.
//
// Defaults to {playlist.ID}_tracks.txt as the filename.
func WriteTextExport(export *models.PlaylistExport, filepath string) (string, error) {
	if filepath == "" {
		filepath = fmt.Sprintf("%s_tracks.txt", export.Playlist.ID)
	}

	textData, err := ExportToText(export)
	if err != nil {
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	if err := os.WriteFile(filepath, textData, 0644); err != nil {
		return "", fmt.Errorf("failed to write text file: %w", err)
	}

	return filepath, nil
}
