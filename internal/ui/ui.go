package ui

import (
	"context"
	"fmt"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/desertthunder/ytx/internal/models"
	"github.com/desertthunder/ytx/internal/services"
	"github.com/desertthunder/ytx/internal/tasks"
)

// ViewState represents the current view in the TUI.
type ViewState int

const (
	PlaylistListView ViewState = iota
	TrackListView
	ConfirmView
	TransferView
	ResultView
)

// Model represents the TUI application state.
type Model struct {
	ctx              context.Context
	view             ViewState
	spotify          services.Service
	engine           *tasks.PlaylistEngine
	width            int
	height           int
	playlistList     list.Model
	playlists        []models.Playlist
	trackList        list.Model
	selectedPlaylist *models.PlaylistExport
	progressChan     chan tasks.ProgressUpdate
	progress         tasks.ProgressUpdate
	result           *tasks.TransferRunResult
	err              error
	help             help.Model
	keys             keyMap
}

// keyMap defines the key bindings for the TUI.
type keyMap struct {
	up       key.Binding
	down     key.Binding
	enter    key.Binding
	back     key.Binding
	yes      key.Binding
	no       key.Binding
	restart  key.Binding
	quit     key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
		back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		yes: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "yes"),
		),
		no: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "no"),
		),
		restart: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "restart"),
		),
		quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.up, k.down, k.enter},
		{k.back, k.yes, k.no},
		{k.restart, k.quit},
	}
}

// playlistItem wraps [models.Playlist] to implement list.Item.
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

// trackItem wraps [models.Track] to implement list.Item.
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

type playlistsFetchedMsg struct {
	playlists []models.Playlist
	err       error
}

type tracksFetchedMsg struct {
	playlist *models.PlaylistExport
	err      error
}

type progressUpdateMsg tasks.ProgressUpdate

type transferCompleteMsg struct {
	result *tasks.TransferRunResult
	err    error
}

// NewModel creates a new TUI model with the provided dependencies.
func NewModel(ctx context.Context, spotify services.Service, engine *tasks.PlaylistEngine) *Model {
	return &Model{
		ctx:     ctx,
		view:    PlaylistListView,
		spotify: spotify,
		engine:  engine,
		help:    help.New(),
		keys:    newKeyMap(),
	}
}

// Init initializes the TUI by fetching playlists from Spotify.
func (m *Model) Init() tea.Cmd {
	return m.fetchPlaylists()
}

// Update handles incoming messages and updates the model state.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.playlistList.Width() == 0 {
			m.playlistList.SetSize(msg.Width-4, msg.Height-8)
		}
		if m.trackList.Width() == 0 {
			m.trackList.SetSize(msg.Width-4, msg.Height-8)
		}
		return m, nil

	case tea.KeyMsg:
		switch m.view {
		case PlaylistListView:
			return m.handlePlaylistListKeys(msg)
		case TrackListView:
			return m.handleTrackListKeys(msg)
		case ConfirmView:
			return m.handleConfirmKeys(msg)
		case ResultView:
			return m.handleResultKeys(msg)
		}

	case playlistsFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tea.Quit
		}
		m.playlists = msg.playlists
		items := make([]list.Item, len(msg.playlists))
		for i, pl := range msg.playlists {
			items[i] = playlistItem{playlist: pl}
		}
		m.playlistList = list.New(items, list.NewDefaultDelegate(), 0, 0)
		m.playlistList.Title = "Spotify Playlists"
		m.playlistList.SetSize(m.width-4, m.height-8)
		return m, nil

	case tracksFetchedMsg:
		if msg.err != nil {
			m.err = msg.err
			m.view = PlaylistListView
			return m, nil
		}
		m.selectedPlaylist = msg.playlist
		items := make([]list.Item, len(msg.playlist.Tracks))
		for i, track := range msg.playlist.Tracks {
			items[i] = trackItem{track: track}
		}
		m.trackList = list.New(items, list.NewDefaultDelegate(), 0, 0)
		m.trackList.Title = fmt.Sprintf("Tracks in '%s'", msg.playlist.Playlist.Name)
		m.trackList.SetSize(m.width-4, m.height-8)
		m.view = TrackListView
		return m, nil

	case progressUpdateMsg:
		m.progress = tasks.ProgressUpdate(msg)
		return m, m.waitForProgress()

	case transferCompleteMsg:
		m.result = msg.result
		m.err = msg.err
		m.view = ResultView
		if m.progressChan != nil {
			close(m.progressChan)
			m.progressChan = nil
		}
		return m, nil
	}

	return m.updateLists(msg)
}

// View renders the UI based on the current view state.
func (m *Model) View() string {
	if m.err != nil && m.view != ResultView {
		return styles.error.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
	}

	switch m.view {
	case PlaylistListView:
		return m.renderPlaylistList()
	case TrackListView:
		return m.renderTrackList()
	case ConfirmView:
		return m.renderConfirm()
	case TransferView:
		return m.renderTransfer()
	case ResultView:
		return m.renderResult()
	default:
		return ""
	}
}

func (m *Model) handlePlaylistListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "enter":
		selected := m.playlistList.SelectedItem()
		if selected != nil {
			if pl, ok := selected.(playlistItem); ok {
				return m, m.fetchTracks(pl.playlist.ID)
			}
		}
	}

	var cmd tea.Cmd
	m.playlistList, cmd = m.playlistList.Update(msg)
	return m, cmd
}

func (m *Model) handleTrackListKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.view = PlaylistListView
		return m, nil
	case "enter":
		m.view = ConfirmView
		return m, nil
	}

	var cmd tea.Cmd
	m.trackList, cmd = m.trackList.Update(msg)
	return m, cmd
}

func (m *Model) handleConfirmKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c", "n":
		m.view = TrackListView
		return m, nil
	case "y":
		m.view = TransferView
		return m, m.startTransfer()
	}
	return m, nil
}

func (m *Model) handleResultKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "r":
		m.view = PlaylistListView
		m.selectedPlaylist = nil
		m.result = nil
		m.err = nil
		return m, nil
	}
	return m, nil
}

func (m *Model) updateLists(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.view {
	case PlaylistListView:
		m.playlistList, cmd = m.playlistList.Update(msg)
	case TrackListView:
		m.trackList, cmd = m.trackList.Update(msg)
	}
	return m, cmd
}

func (m *Model) fetchPlaylists() tea.Cmd {
	return func() tea.Msg {
		playlists, err := m.spotify.GetPlaylists(m.ctx)
		return playlistsFetchedMsg{playlists: playlists, err: err}
	}
}

func (m *Model) fetchTracks(playlistID string) tea.Cmd {
	return func() tea.Msg {
		playlist, err := m.spotify.ExportPlaylist(m.ctx, playlistID)
		return tracksFetchedMsg{playlist: playlist, err: err}
	}
}

func (m *Model) startTransfer() tea.Cmd {
	m.progressChan = make(chan tasks.ProgressUpdate, 50)

	go func() {
		result, err := m.engine.Run(m.ctx, m.selectedPlaylist.Playlist.ID, m.progressChan)
		m.result = result
		m.err = err
		close(m.progressChan)
	}()

	return m.waitForProgress()
}

func (m *Model) waitForProgress() tea.Cmd {
	return func() tea.Msg {
		if m.progressChan == nil {
			return transferCompleteMsg{result: m.result, err: m.err}
		}

		update, ok := <-m.progressChan
		if !ok {
			return transferCompleteMsg{result: m.result, err: m.err}
		}
		return progressUpdateMsg(update)
	}
}

func (m *Model) renderPlaylistList() string {
	helpKeys := []key.Binding{m.keys.enter, m.keys.quit}
	helpView := m.help.ShortHelpView(helpKeys)
	return fmt.Sprintf("%s\n\n%s", m.playlistList.View(), helpView)
}

func (m *Model) renderTrackList() string {
	transferKey := key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "transfer"),
	)
	helpKeys := []key.Binding{transferKey, m.keys.back, m.keys.quit}
	helpView := m.help.ShortHelpView(helpKeys)
	return fmt.Sprintf("%s\n\n%s", m.trackList.View(), helpView)
}

func (m *Model) renderConfirm() string {
	title := styles.title.Render(fmt.Sprintf("Transfer '%s' to YouTube Music?", m.selectedPlaylist.Playlist.Name))
	info := fmt.Sprintf("\nPlaylist: %s\nTracks: %d\n", m.selectedPlaylist.Playlist.Name, len(m.selectedPlaylist.Tracks))

	helpKeys := []key.Binding{m.keys.yes, m.keys.no, m.keys.quit}
	helpView := m.help.ShortHelpView(helpKeys)

	return fmt.Sprintf("%s\n%s\n%s", title, info, helpView)
}

func (m *Model) renderTransfer() string {
	title := styles.title.Render("Transferring Playlist")

	var phase string
	switch m.progress.Phase {
	case tasks.FetchSource:
		phase = "Fetching source playlist..."
	case tasks.SearchTracks:
		phase = fmt.Sprintf("Searching tracks (%d/%d)", m.progress.Step, m.progress.Total)
	case tasks.CreatePlaylist:
		phase = "Creating playlist on YouTube Music..."
	default:
		phase = "Processing..."
	}

	return fmt.Sprintf("%s\n\n%s\n%s", title, phase, m.progress.Message)
}

func (m *Model) renderResult() string {
	if m.err != nil {
		return styles.error.Render(fmt.Sprintf("Transfer failed: %v\n\nPress r to retry, q to quit", m.err))
	}

	if m.result == nil {
		return styles.error.Render("No result available\n\nPress r to retry, q to quit")
	}

	title := styles.success.Render("✓ Transfer Complete!")
	info := fmt.Sprintf(
		"\nSource: %s (%d tracks)\nDestination: %s (%d tracks)\nSuccess rate: %d/%d (%.1f%%)",
		m.result.SourcePlaylist.Playlist.Name,
		m.result.TotalTracks,
		m.result.DestPlaylist.Name,
		m.result.SuccessCount,
		m.result.SuccessCount,
		m.result.TotalTracks,
		m.result.MatchPercentage,
	)

	var failed string
	if m.result.FailedCount > 0 {
		failed = fmt.Sprintf("\n\n%s", styles.warning.Render(fmt.Sprintf("Failed to match %d tracks:", m.result.FailedCount)))
		for _, match := range m.result.TrackMatches {
			if match.Error != nil {
				failed += fmt.Sprintf("\n  • %s - %s", match.Original.Artist, match.Original.Title)
			}
		}
	}

	helpKeys := []key.Binding{m.keys.restart, m.keys.quit}
	helpView := m.help.ShortHelpView(helpKeys)

	return fmt.Sprintf("%s\n%s%s\n\n%s", title, info, failed, helpView)
}
