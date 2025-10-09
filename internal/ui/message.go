package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/tasks"
)

// MsgKind enumerates all message types in the application.
type MsgKind int

// Msg represents all possible messages in the TUI (Elm-style message union).
type Msg struct {
	kind MsgKind
	data any
}

var (
	_ tea.Msg = Msg{}
)

const (
	MsgPlaylistsFetched MsgKind = iota
	MsgTracksFetched
	MsgProgressUpdate
	MsgTransferComplete
)

// playlistsFetchedMsg is the constructor for [MsgPlaylistsFetched]
func playlistsFetchedMsg(playlists []models.Playlist, err error) Msg {
	return Msg{
		kind: MsgPlaylistsFetched,
		data: struct {
			playlists []models.Playlist
			err       error
		}{playlists, err},
	}
}

// tracksFetchedMsg is the constructor for [MsgTracksFetched]
func tracksFetchedMsg(playlist *models.PlaylistExport, err error) Msg {
	return Msg{
		kind: MsgTracksFetched,
		data: struct {
			playlist *models.PlaylistExport
			err      error
		}{playlist, err},
	}
}

// progressUpdateMsg is the constructor for [MsgProgressUpdate]
func progressUpdateMsg(update tasks.ProgressUpdate) Msg {
	return Msg{kind: MsgProgressUpdate, data: update}
}

// transferCompleteMsg is the constructor for [MsgTransferComplete]
func transferCompleteMsg(result *tasks.TransferRunResult, err error) Msg {
	return Msg{
		kind: MsgTransferComplete,
		data: struct {
			result *tasks.TransferRunResult
			err    error
		}{result, err},
	}
}
