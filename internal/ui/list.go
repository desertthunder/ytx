package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	"github.com/desertthunder/ytx/internal/models"
)

var (
	_ list.Item = playlistItem{}
	_ list.Item = trackItem{}
)

// playlistItem wraps [models.Playlist] to implement [list.Item].
type playlistItem struct {
	playlist models.Playlist
}

func (i playlistItem) FilterValue() string { return i.playlist.Name }
func (i playlistItem) Title() string       { return i.playlist.Name }
func (i playlistItem) Description() string {
	desc := fmt.Sprintf("%d tracks", i.playlist.TrackCount)
	if i.playlist.Description != "" {
		desc = fmt.Sprintf("%s • %s", desc, i.playlist.Description)
	}
	return desc
}

// trackItem wraps [models.Track] to implement [list.Item].
type trackItem struct {
	track models.Track
}

func (i trackItem) FilterValue() string { return i.track.Title }
func (i trackItem) Title() string       { return i.track.Title }
func (i trackItem) Description() string {
	desc := i.track.Artist
	if i.track.Album != "" {
		desc = fmt.Sprintf("%s • %s", desc, i.track.Album)
	}
	return desc
}
