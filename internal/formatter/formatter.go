// package formatter provides functions to export playlist data to various formats (CSV, Markdown, plain text)
package formatter

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/shared"
)

// CSVExportResult contains the paths of files created by WriteCSVExport
type CSVExportResult struct {
	TracksFile   string
	MetadataFile string
}

// MarkdownExportResult contains information about files created by WriteMarkdownExport
type MarkdownExportResult struct {
	Directory  string
	Files      []string
	CoverImage string
}

// ExportManifest represents a summary of a bulk export operation.
type ExportManifest struct {
	Timestamp         string                `json:"timestamp"`
	Format            string                `json:"format"`
	TotalPlaylists    int                   `json:"total_playlists"`
	SuccessfulExports int                   `json:"successful_exports"`
	FailedExports     int                   `json:"failed_exports"`
	Exports           []ExportManifestEntry `json:"exports"`
}

// ExportManifestEntry represents a single playlist export in the manifest.
type ExportManifestEntry struct {
	PlaylistID   string   `json:"playlist_id"`
	PlaylistName string   `json:"playlist_name"`
	Status       string   `json:"status"`
	Files        []string `json:"files,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// Type assertion helper struct matching tasks.BulkExportResult for JSON unmarshaling
type BulkExportResult struct {
	TotalPlaylists    int
	SuccessfulExports int
	FailedExports     int
	Results           []struct {
		PlaylistID   string
		PlaylistName string
		Success      bool
		Files        []string
		Error        interface{} // Use interface{} to handle both error objects and strings
	}
	OutputDirectory string
	ManifestPath    string
}

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

// ExportToJSON converts a PlaylistExport to JSON format
func ExportToJSON(export *models.PlaylistExport) ([]byte, error) {
	return shared.MarshalJSON(export, true)
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

// WriteJSONExport exports a playlist to JSON format.
//
// Defaults to {playlist.ID}.json as the filename.
func WriteJSONExport(export *models.PlaylistExport, filepath string) (string, error) {
	if filepath == "" {
		filepath = fmt.Sprintf("%s.json", export.Playlist.ID)
	}

	jsonData, err := ExportToJSON(export)
	if err != nil {
		return "", fmt.Errorf("failed to generate JSON: %w", err)
	}

	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return "", fmt.Errorf("failed to write JSON file: %w", err)
	}

	return filepath, nil
}

// WriteBulkExportManifest writes a JSON manifest file summarizing bulk export results.
// The manifest includes timestamp, format, success/failure counts, and per-playlist details.
// Accepts any result type with matching structure via JSON marshaling.
func WriteBulkExportManifest(result any, format string, filepath string) error {
	// Use JSON marshaling/unmarshaling to convert from any compatible type
	jsonData, err := shared.MarshalJSON(result, false)
	if err != nil {
		return fmt.Errorf("failed to marshal input result: %w", err)
	}

	var bulkResult BulkExportResult
	if err := json.Unmarshal(jsonData, &bulkResult); err != nil {
		return fmt.Errorf("invalid result type for manifest: %w", err)
	}

	manifest := ExportManifest{
		Timestamp:         time.Now().UTC().Format(time.RFC3339),
		Format:            format,
		TotalPlaylists:    bulkResult.TotalPlaylists,
		SuccessfulExports: bulkResult.SuccessfulExports,
		FailedExports:     bulkResult.FailedExports,
		Exports:           make([]ExportManifestEntry, 0, len(bulkResult.Results)),
	}

	for _, res := range bulkResult.Results {
		entry := ExportManifestEntry{
			PlaylistID:   res.PlaylistID,
			PlaylistName: res.PlaylistName,
			Files:        res.Files,
		}

		if res.Success {
			entry.Status = "success"
		} else {
			entry.Status = "failed"
			if res.Error != nil {
				// Convert error (could be string, map, or other types from JSON)
				if errStr, ok := res.Error.(string); ok {
					entry.Error = errStr
				} else if errMap, ok := res.Error.(map[string]interface{}); ok {
					// Error marshaled as object, extract message
					if msg, ok := errMap["Error"].(string); ok {
						entry.Error = msg
					} else {
						entry.Error = fmt.Sprintf("%v", res.Error)
					}
				} else {
					entry.Error = fmt.Sprintf("%v", res.Error)
				}
			}
		}

		manifest.Exports = append(manifest.Exports, entry)
	}

	data, err := shared.MarshalJSON(manifest, true)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest file: %w", err)
	}
	return nil
}
