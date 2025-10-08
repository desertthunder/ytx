// Package ui implements an interactive terminal interface using bubbletea's Elm architecture.
//
// The TUI provides a multi-view workflow for playlist migration:
//  1. [PlaylistListView] : Browse and select Spotify playlists
//  2. [TrackListView] : Preview tracks before transfer
//  3. [ConfirmView] : Confirm transfer operation
//  4. [TransferView] : Monitor real-time progress updates
//  5. [ResultView] : Display success metrics and failed matches
//
// The (view) [Model] implements bubbletea/Elm's standard Init/Update/View pattern, receiving messages via the Msg union type.
// Progress updates flow through a channel from the PlaylistEngine, providing non-blocking status reporting during transfers.
//
// Keyboard navigation uses vim-style bindings (j/k, enter, esc, y/n, q) with contextual help displayed via charmbracelet/bubbles/help.
package ui
