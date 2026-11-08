package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dhowden/tag"
	_ "golang.org/x/image/webp"

	"MusicLe/bridge"
	"MusicLe/internal/browser"
	"MusicLe/state"
	"MusicLe/ui"

	_ "image/jpeg"
	_ "image/png"
)

type HomeModel struct {
	width  int
	height int
	ready  bool

	focusIdx     int
	sectionFocus int // 0=sidebar(console), 1=playlist, 2=songs

	songFocusIdx    int
	songActionFocus int // 0=play, 1=edit, 2=delete, -1=none
	songOffset      int
	bodyHeight      int

	songEndedAt  time.Time
	manualStop   bool
	shuffleOrder []int
	shufflePos   int

	playlistOptions     []string
	playlistIdx         int
	playlistExpanded    bool
	playlistActionFocus int // 0=name preview, 1=Play All, 2=Shuffle, -1=none
	previewIdx          int // playlist index being previewed (-1 = using playlistIdx)

	logLines      []string
	consoleScroll int

	// Browser connector cards on the home sidebar (below the console)
	connectFocus int // 0 = Spotify, 1 = YouTube Music
	sidebarFocus int // 0 = console, 1 = connect cards

	editModalOpen bool
	editSongIdx   int
	editTitle     textinput.Model
	editArtist    textinput.Model
	editDuration  textinput.Model
	editFocus     int

	deleteConfirm bool
	deleteSongIdx int
	deleteYes     bool

	renameMode  bool
	renameInput textinput.Model

	editSelectAll   bool
	renameSelectAll bool

	smoothBands [17]float64
	fadeLevel   float64 // 0-1, global fade for spectrum (smooth on play/pause)

	// Now-playing cover cache: decoded once per song, shown in the card's
	// art area. Empty path + nil image = nothing playing or no artwork.
	coverPath string
	coverImg  image.Image
}

func NewHomeModel() *HomeModel {
	return &HomeModel{
		fadeLevel:           1,
		playlistIdx:         0,
		sectionFocus:        -1,
		playlistActionFocus: -1,
		previewIdx:          -1,
		songFocusIdx:        -1,
		songActionFocus:     -1,
		consoleScroll:       -1,
		editTitle:           editInput(langT("Title", "Baslik")),
		editArtist:          editInput(langT("Artist", "Sanatci")),
		editDuration:        editInput(langT("Duration", "Sure")),
		renameInput: func() textinput.Model {
			ti := textinput.New()
			ti.Prompt = ""
			ti.Width = 30
			ti.CharLimit = 100
			ti.Cursor.Style = lipgloss.NewStyle().
				Background(ui.ColorAccent).
				Foreground(lipgloss.Color("#000000"))
			return ti
		}(),
	}
}

func editInput(prompt string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = "  " + prompt + ":  "
	ti.Cursor.Style = lipgloss.NewStyle().
		Background(lipgloss.Color("#1DB954")).
		Foreground(lipgloss.Color("#000000"))
	ti.Width = 50
	ti.CharLimit = 100
	return ti
}

func (m *HomeModel) Init() tea.Cmd {
	m.refreshPlaylistOptions()
	m.focusIdx = -1
	return nil
}

func (m *HomeModel) refreshPlaylistOptions() {
	m.playlistOptions = nil
	if state.Current.CurrentProfile != nil {
		for _, pl := range state.Current.CurrentProfile.Playlists {
			m.playlistOptions = append(m.playlistOptions, pl.Name)
		}
	}
	if len(m.playlistOptions) == 0 {
		m.playlistOptions = []string{Tr("dl.no_playlists")}
	}
	if m.playlistIdx >= len(m.playlistOptions) {
		m.playlistIdx = 0
	}
}

func (m *HomeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		if m.editModalOpen {
			return m.handleEditModalKey(msg)
		}
		if m.deleteConfirm {
			return m.handleDeleteKey(msg)
		}
		if m.renameMode {
			return m.handleRenameKey(msg)
		}
		return m.handleKeyMsg(msg)

	case PlayerStatusResult:
		if msg.Result != nil {
			m.processPlayerStatus(msg.Result)
		}

	case PlayResultMsg:
		if msg.Error != nil {
			m.addLog("error", fmt.Sprintf("Play error: %s - %v", msg.Title, msg.Error))
		} else {
			m.addLog("ok", fmt.Sprintf("Playing: %s", msg.Title))
		}

	case ImportResultMsg:
		return m, m.handleImportResult(msg)

	case deleteSongResultMsg:
		if msg.errMsg != "" {
			m.addLog("error", "Delete failed: "+msg.errMsg)
			return m, nil
		}
		if cur := state.Current.Player.CurrentSong; cur != nil && cur.FilePath == msg.filePath {
			state.Current.Player.CurrentSong = nil
			state.Current.Player.IsPlaying = false
			state.Current.Player.IsPaused = false
		}
		m.refreshAllContent()
		m.clampSongFocus()
		m.addLog("ok", "Deleted: "+msg.title)
		return m, nil

	case PlaySongMsg:
		pl := state.Current.CurrentPlaylist
		if pl != nil {
			for i := range pl.Songs {
				if pl.Songs[i].FilePath == msg.FilePath {
					m.songFocusIdx = i
					// Adjust songOffset so selected song is visible
					rows := m.bodyHeight - 10
					if rows < 1 {
						rows = 1
					}
					if m.songFocusIdx < m.songOffset {
						m.songOffset = m.songFocusIdx
					} else if m.songFocusIdx >= m.songOffset+rows {
						m.songOffset = m.songFocusIdx - rows + 1
					}
					// Auto-advance: keep the running shuffle session intact.
					return m, m.startSong(&pl.Songs[i])
				}
			}
		}
	}

	return m, nil
}

func (m *HomeModel) songs() []state.Song {
	if state.Current.CurrentPlaylist != nil {
		return state.Current.CurrentPlaylist.Songs
	}
	return nil
}

func (m *HomeModel) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.playlistExpanded {
		return m.handlePlaylistKey(msg)
	}

	// Browser connector cards on the home sidebar
	if m.sectionFocus == 0 && m.sidebarFocus == 1 {
		switch msg.String() {
		case "left", "up", "shift+tab":
			m.connectFocus = 0
			return m, nil
		case "right", "down", "tab":
			m.connectFocus = 1
			return m, nil
		case "enter", " ":
			return m.launchConnectFromHome()
		}
	}

	switch msg.String() {
	case "tab":
		if m.sectionFocus == 0 {
			m.sidebarFocus = (m.sidebarFocus + 1) % 2
			return m, nil
		}
		if m.sectionFocus == 1 {
			if pl := state.Current.CurrentPlaylist; pl != nil && len(pl.Songs) > 0 {
				m.playlistActionFocus = (m.playlistActionFocus + 1) % 3
				if m.playlistActionFocus == 0 {
					m.previewIdx = m.playlistIdx
				}
			}
		} else if m.focusIdx == 6 {
			songs := m.songs()
			if len(songs) > 0 {
				oldFocus := m.songFocusIdx
				m.songFocusIdx = (m.songFocusIdx + 1) % len(songs)
				m.songActionFocus = 0
				if m.songFocusIdx == 0 && oldFocus == len(songs)-1 {
					m.songOffset = 0
				} else {
					maxVis := m.maxVisibleSongs()
					if m.songFocusIdx >= m.songOffset+maxVis {
						m.songOffset = m.songFocusIdx - maxVis + 1
					}
				}
			}
		}
		return m, nil
	case "shift+tab":
		if m.sectionFocus == 0 {
			m.sidebarFocus = (m.sidebarFocus + 1) % 2
			return m, nil
		}
		if m.sectionFocus == 1 {
			if pl := state.Current.CurrentPlaylist; pl != nil && len(pl.Songs) > 0 {
				m.playlistActionFocus = (m.playlistActionFocus - 1 + 3) % 3
				if m.playlistActionFocus == 0 {
					m.previewIdx = m.playlistIdx
				}
			}
		} else if m.focusIdx == 6 {
			songs := m.songs()
			if len(songs) > 0 {
				oldFocus := m.songFocusIdx
				m.songFocusIdx = (m.songFocusIdx - 1 + len(songs)) % len(songs)
				m.songActionFocus = 0
				if m.songFocusIdx == len(songs)-1 && oldFocus == 0 {
					m.songOffset = len(songs) - m.maxVisibleSongs()
					if m.songOffset < 0 {
						m.songOffset = 0
					}
				} else if m.songFocusIdx < m.songOffset {
					m.songOffset = m.songFocusIdx
				}
			}
		}
		return m, nil
	case "enter":
		return m.handleEnter()
	case " ":
		m.togglePlayPause()
		return m, nil
	case "right":
		if m.focusIdx == 6 && m.songFocusIdx >= 0 {
			m.songActionFocus = (m.songActionFocus + 1) % 3
			return m, tea.HideCursor
		}
		go bridge.PlayerCall(bridge.Action{Action: "seek", Value: 5})
		return m, nil
	case "left":
		if m.focusIdx == 6 && m.songFocusIdx >= 0 {
			m.songActionFocus = (m.songActionFocus - 1 + 3) % 3
			return m, tea.HideCursor
		}
		go bridge.PlayerCall(bridge.Action{Action: "seek", Value: -5})
		return m, nil
	case "f7":
		return m, m.playSelectedSong()
	case "f5":
		m.applyRegion(0)
		return m, tea.HideCursor
	case "f6":
		m.applyRegion(4)
		return m, tea.HideCursor
	case "e":
		if m.focusIdx == 6 && m.songFocusIdx >= 0 {
			m.openEditModal()
			return m, nil
		}
	case "d", "delete":
		if m.focusIdx == 6 && m.songFocusIdx >= 0 {
			m.openDeleteConfirm()
			return m, nil
		}
	case "r":
		if m.sectionFocus == 1 {
			m.startRename()
			return m, nil
		}
	case "up":
		if m.sectionFocus == 0 {
			if m.consoleScroll < 0 {
				m.consoleScroll = len(m.logLines)
			}
			if m.consoleScroll > 0 {
				m.consoleScroll--
			}
			return m, nil
		}
		if m.sectionFocus == 1 {
			plen := len(m.playlistOptions)
			if plen > 0 {
				if m.playlistActionFocus == 0 {
					m.previewIdx = (m.previewIdx - 1 + plen) % plen
				} else {
					m.playlistIdx = (m.playlistIdx - 1 + plen) % plen
					m.selectPlaylist(m.playlistIdx)
				}
			}
			return m, nil
		}
		if m.focusIdx == 6 {
			songs := m.songs()
			if len(songs) > 0 {
				oldFocus := m.songFocusIdx
				m.songFocusIdx = (m.songFocusIdx - 1 + len(songs)) % len(songs)
				m.songActionFocus = 0
				if m.songFocusIdx == len(songs)-1 && oldFocus == 0 {
					m.songOffset = len(songs) - m.maxVisibleSongs()
					if m.songOffset < 0 {
						m.songOffset = 0
					}
				} else if m.songFocusIdx < m.songOffset {
					m.songOffset = m.songFocusIdx
				}
			}
			return m, nil
		}
		m.adjustVolume(0.05)
		return m, nil
	case "down":
		if m.sectionFocus == 0 {
			if m.consoleScroll < 0 {
				return m, nil
			}
			m.consoleScroll++
			return m, nil
		}
		if m.sectionFocus == 1 {
			plen := len(m.playlistOptions)
			if plen > 0 {
				if m.playlistActionFocus == 0 {
					m.previewIdx = (m.previewIdx + 1) % plen
				} else {
					m.playlistIdx = (m.playlistIdx + 1) % plen
					m.selectPlaylist(m.playlistIdx)
				}
			}
			return m, nil
		}
		if m.focusIdx == 6 {
			songs := m.songs()
			if len(songs) > 0 {
				oldFocus := m.songFocusIdx
				m.songFocusIdx = (m.songFocusIdx + 1) % len(songs)
				m.songActionFocus = 0
				if m.songFocusIdx == 0 && oldFocus == len(songs)-1 {
					m.songOffset = 0
				} else {
					maxVis := m.maxVisibleSongs()
					if m.songFocusIdx >= m.songOffset+maxVis {
						m.songOffset = m.songFocusIdx - maxVis + 1
					}
				}
			}
			return m, nil
		}
		m.adjustVolume(-0.05)
		return m, nil
	case "pgup":
		if m.sectionFocus == 0 {
			if m.consoleScroll < 0 {
				m.consoleScroll = len(m.logLines)
			}
			if m.consoleScroll > 0 {
				m.consoleScroll -= 10
				if m.consoleScroll < 0 {
					m.consoleScroll = 0
				}
			}
			return m, nil
		}
		return m, nil
	case "pgdown":
		if m.sectionFocus == 0 {
			if m.consoleScroll < 0 {
				return m, nil
			}
			m.consoleScroll += 10
			return m, nil
		}
		return m, nil
	case "end":
		if m.sectionFocus == 0 {
			m.consoleScroll = -1
			return m, nil
		}
		return m, nil
	case "home":
		if m.sectionFocus == 0 {
			m.consoleScroll = 0
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

func (m *HomeModel) handlePlaylistKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		plen := len(m.playlistOptions)
		if plen > 0 {
			m.playlistIdx = (m.playlistIdx - 1 + plen) % plen
		}
	case "down", "j":
		plen := len(m.playlistOptions)
		if plen > 0 {
			m.playlistIdx = (m.playlistIdx + 1) % plen
		}
	case "enter":
		m.selectPlaylist(m.playlistIdx)
		m.playlistExpanded = false
	case "esc":
		m.playlistExpanded = false
	case "tab":
		m.playlistExpanded = false
	case "shift+tab":
		m.playlistExpanded = false
	}
	return m, nil
}

// currentHomeRegion derives the active home focus region from the focus fields:
// 0=console, 1=Spotify card, 2=YouTube card, 3=playlist, 4=songs.
func (m *HomeModel) currentHomeRegion() int {
	switch {
	case m.sidebarFocus == 1 && m.connectFocus == 0:
		return 1
	case m.sidebarFocus == 1 && m.connectFocus == 1:
		return 2
	case m.sectionFocus == 1:
		return 3
	case m.sectionFocus == 2:
		return 4
	}
	return 0
}

// applyRegion sets the focus fields for a given home region (0..4).
func (m *HomeModel) applyRegion(r int) {
	m.songFocusIdx = -1
	m.songActionFocus = -1
	m.focusIdx = -1
	m.editSelectAll = false
	m.playlistActionFocus = -1
	m.previewIdx = -1
	switch r {
	case 0: // console
		m.sidebarFocus = 0
		m.connectFocus = 0
		m.sectionFocus = 0
	case 1: // Spotify card
		m.sidebarFocus = 1
		m.connectFocus = 0
		m.sectionFocus = 0
	case 2: // YouTube Music card
		m.sidebarFocus = 1
		m.connectFocus = 1
		m.sectionFocus = 0
	case 3: // playlist
		m.sidebarFocus = 0
		m.connectFocus = 0
		m.sectionFocus = 1
		m.playlistActionFocus = 0
	case 4: // songs
		m.sidebarFocus = 0
		m.connectFocus = 0
		m.sectionFocus = 2
		m.focusIdx = 6
		if songs := m.songs(); len(songs) > 0 {
			m.songFocusIdx = 0
			m.songActionFocus = 0
		}
	}
}

// CycleSection implements the F1 focus cycle on the home screen:
// bottom bar -> Spotify -> YouTube Music -> playlist -> songs -> (wrap) bottom bar.
// The console (live output) is intentionally not part of the F1 cycle.
func (m *HomeModel) CycleSection() (bool, tea.Cmd) {
	r := m.currentHomeRegion()
	if r < 1 {
		r = 1
	}
	if r >= 4 {
		// Wrapped past the last region: hand focus back to the bottom bar.
		m.sidebarFocus = 0
		m.connectFocus = 0
		m.sectionFocus = -1
		m.songFocusIdx = -1
		m.songActionFocus = -1
		m.focusIdx = -1
		m.playlistActionFocus = -1
		return true, tea.HideCursor
	}
	m.applyRegion(r + 1)
	return false, tea.HideCursor
}

func (m *HomeModel) maxVisibleSongs() int {
	h := m.bodyHeight
	if h < 10 {
		h = m.height
	}
	// tableBox = 2(border) + 1(title) + 1(sep) + 3(header) + 3*N(songs) + 1(sep) + 1(hint)
	// total = 9 + 3*N. Fit within h: N = (h - 9) / 3
	n := (h - 9) / 3
	if n < 1 {
		n = 1
	}
	return n
}

func (m *HomeModel) editCurrentInput() *textinput.Model {
	switch m.editFocus {
	case 0:
		return &m.editTitle
	case 1:
		return &m.editArtist
	case 2:
		return &m.editDuration
	}
	return nil
}

func (m *HomeModel) handleEnter() (tea.Model, tea.Cmd) {
	if m.sectionFocus == 1 {
		switch m.playlistActionFocus {
		case 0:
			if m.previewIdx >= 0 {
				m.playlistIdx = m.previewIdx
				m.selectPlaylist(m.playlistIdx)
				m.previewIdx = -1
			}
		case 1:
			return m, m.playAllSongs()
		case 2:
			return m, m.playShuffledSongs()
		}
		return m, tea.HideCursor
	}
	if m.focusIdx == 6 {
		songs := m.songs()
		if m.songFocusIdx >= 0 && m.songFocusIdx < len(songs) {
			switch m.songActionFocus {
			case 0:
				return m, m.playSong(&songs[m.songFocusIdx])
			case 1:
				m.openEditModal()
			case 2:
				m.openDeleteConfirm()
			}
		}
	}
	return m, tea.HideCursor
}

func (m *HomeModel) handleImportResult(msg ImportResultMsg) tea.Cmd {
	if msg.Error != nil || msg.Result.Status == "error" {
		errMsg := ""
		if msg.Result != nil {
			errMsg = msg.Result.Error
		}
		if errMsg == "" && msg.Error != nil {
			errMsg = msg.Error.Error()
		}
		m.addLog("error", "Import failed: "+errMsg)
		return nil
	}
	m.addLog("ok", "Imported: "+msg.Result.Filename)
	m.refreshAllContent()
	return nil
}

func (m *HomeModel) openEditModal() {
	songs := m.songs()
	idx := m.songFocusIdx
	if idx < 0 || idx >= len(songs) {
		return
	}
	m.editSongIdx = idx
	song := songs[idx]
	m.editTitle.SetValue(song.Title)
	m.editArtist.SetValue(song.Artist)
	m.editDuration.SetValue(normalizeDuration(song.Duration))
	m.editFocus = 0
	m.editTitle.Focus()
	m.editModalOpen = true
}

func (m *HomeModel) handleEditModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeEditModal()
		return m, nil
	case "tab":
		m.editFocus = (m.editFocus + 1) % 3
		m.setEditFocus()
		return m, nil
	case "shift+tab":
		m.editFocus = (m.editFocus - 1 + 3) % 3
		m.setEditFocus()
		return m, nil
	case "enter":
		return m.saveEditModal()
	case "ctrl+v":
		var cmd tea.Cmd
		switch m.editFocus {
		case 0:
			m.editTitle, cmd = m.editTitle.Update(textinput.Paste())
		case 1:
			m.editArtist, cmd = m.editArtist.Update(textinput.Paste())
		case 2:
			m.editDuration, cmd = m.editDuration.Update(textinput.Paste())
		}
		return m, cmd
	case "ctrl+a":
		inp := m.editCurrentInput()
		if inp != nil && inp.Value() != "" {
			m.editSelectAll = true
		}
		return m, nil
	}
	if m.editSelectAll {
		inp := m.editCurrentInput()
		if inp != nil {
			s := msg.String()
			if len(s) == 1 || s == "backspace" || s == "delete" {
				inp.SetValue("")
				inp.SetCursor(0)
				m.editSelectAll = false
			} else {
				m.editSelectAll = false
			}
		}
	}
	switch m.editFocus {
	case 0:
		var cmd tea.Cmd
		m.editTitle, cmd = m.editTitle.Update(msg)
		return m, cmd
	case 1:
		var cmd tea.Cmd
		m.editArtist, cmd = m.editArtist.Update(msg)
		return m, cmd
	case 2:
		var cmd tea.Cmd
		m.editDuration, cmd = m.editDuration.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *HomeModel) setEditFocus() {
	m.editTitle.Blur()
	m.editArtist.Blur()
	m.editDuration.Blur()
	switch m.editFocus {
	case 0:
		m.editTitle.Focus()
	case 1:
		m.editArtist.Focus()
	case 2:
		m.editDuration.Focus()
	}
}

func (m *HomeModel) closeEditModal() {
	m.editModalOpen = false
	m.editTitle.Blur()
	m.editArtist.Blur()
	m.editDuration.Blur()
}

func (m *HomeModel) saveEditModal() (tea.Model, tea.Cmd) {
	songs := m.songs()
	if m.editSongIdx < 0 || m.editSongIdx >= len(songs) {
		m.closeEditModal()
		return m, nil
	}
	song := songs[m.editSongIdx]
	title := strings.TrimSpace(m.editTitle.Value())
	artist := strings.TrimSpace(m.editArtist.Value())
	duration := strings.TrimSpace(m.editDuration.Value())
	if title == "" {
		title = song.Title
	}
	if artist == "" {
		artist = song.Artist
	}
	if duration == "" {
		duration = song.Duration
	}
	duration = normalizeDuration(duration)
	pl := state.Current.CurrentPlaylist
	if pl == nil {
		m.closeEditModal()
		return m, nil
	}
	profile := state.Current.CurrentProfile
	if profile == nil {
		m.closeEditModal()
		return m, nil
	}
	listPath := state.Current.SongListPath(profile.FolderName, pl.FolderName)

	m.closeEditModal()

	return m, func() tea.Msg {
		result, err := bridge.RunScript(bridge.Action{
			Action: "update_song",
			File:   listPath,
			Path:   song.FilePath,
			Value:  map[string]string{"title": title, "artist": artist, "duration": duration},
		})
		if err == nil && result.Status == "ok" {
			_ = state.Current.ScanProfiles()
			m.refreshAllContent()
			m.addLog("ok", "Updated: "+song.Title)
		} else {
			errMsg := ""
			if err != nil {
				errMsg = err.Error()
			} else if result != nil {
				errMsg = result.Error
			}
			m.addLog("error", "Update failed: "+errMsg)
		}
		return nil
	}
}

func (m *HomeModel) renderEditOverlay(full string) string {
	titleLbl := ui.AccentStyle.Render(" "+Tr("home.title")+" ") + "\n" + m.editTitle.View()
	artistLbl := ui.AccentStyle.Render(" "+Tr("home.artist")+" ") + "\n" + m.editArtist.View()
	durLbl := ui.AccentStyle.Render(" "+Tr("home.duration")+" ") + "\n" + m.editDuration.View()
	content := lipgloss.JoinVertical(lipgloss.Left, titleLbl, "", artistLbl, "", durLbl, "", ui.DimStyle.Render(Tr("home.edit_hint")))
	content = ui.BorderStyle.Width(50).Render(ui.WhiteStyle.Bold(true).Render(Tr("home.edit_title")) + "\n" + content)
	return m.placeOverlay(full, content)
}

func (m *HomeModel) placeOverlay(full, overlay string) string {
	lines := strings.Split(full, "\n")
	totalH := len(lines)
	contentH := lipgloss.Height(overlay)
	contentW := lipgloss.Width(overlay)
	topPad := (totalH - contentH) / 2
	leftPad := (m.width - contentW) / 2
	if topPad < 0 {
		topPad = 0
	}
	if leftPad < 0 {
		leftPad = 0
	}
	overlayLines := strings.Split(overlay, "\n")
	var result []string
	for i, line := range lines {
		if i >= topPad && i < topPad+contentH {
			ci := i - topPad
			if ci >= 0 && ci < len(overlayLines) {
				result = append(result, strings.Repeat(" ", leftPad)+overlayLines[ci])
			} else {
				result = append(result, line)
			}
		} else {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

func (m *HomeModel) openDeleteConfirm() {
	songs := m.songs()
	idx := m.songFocusIdx
	if idx < 0 || idx >= len(songs) {
		return
	}
	m.deleteSongIdx = idx
	m.deleteYes = false
	m.deleteConfirm = true
}

// inputCaptured reports whether a home overlay owns the keyboard. While one is
// open the global key handlers (player bar, F1/F2/F3, view switching) must not
// swallow the keys the overlay is waiting for.
func (m *HomeModel) inputCaptured() bool {
	return m.deleteConfirm || m.editModalOpen || m.renameMode || m.playlistExpanded
}

// clampSongFocus keeps the song selection inside the current list after the
// list shrinks (song deleted) or changes (playlist switched).
func (m *HomeModel) clampSongFocus() {
	n := len(m.songs())
	if m.songFocusIdx >= n {
		m.songFocusIdx = n - 1
	}
	if m.songOffset > m.songFocusIdx {
		m.songOffset = m.songFocusIdx
	}
	if m.songOffset < 0 {
		m.songOffset = 0
	}
}

func (m *HomeModel) handleDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.deleteConfirm = false
		return m, nil
	case "left", "right", "up", "down", "tab", "shift+tab", " ":
		m.deleteYes = !m.deleteYes
		return m, nil
	case "enter":
		return m.executeDelete()
	}
	return m, nil
}

func (m *HomeModel) executeDelete() (tea.Model, tea.Cmd) {
	if !m.deleteYes {
		m.deleteConfirm = false
		return m, nil
	}
	songs := m.songs()
	if m.deleteSongIdx < 0 || m.deleteSongIdx >= len(songs) {
		m.deleteConfirm = false
		return m, nil
	}
	song := songs[m.deleteSongIdx]
	pl := state.Current.CurrentPlaylist
	if pl == nil {
		m.deleteConfirm = false
		return m, nil
	}
	profile := state.Current.CurrentProfile
	if profile == nil {
		m.deleteConfirm = false
		return m, nil
	}
	listPath := state.Current.SongListPath(profile.FolderName, pl.FolderName)
	isCurrent := false
	if cur := state.Current.Player.CurrentSong; cur != nil && cur.FilePath == song.FilePath {
		isCurrent = true
	}

	m.deleteConfirm = false

	// The command only talks to disk/audio; the model is updated in Update
	// when deleteSongResultMsg comes back, so the UI repaints with the new
	// library instead of waiting for the next poll tick.
	return m, func() tea.Msg {
		if isCurrent {
			_, _ = bridge.PlayerCall(bridge.Action{Action: "stop"})
		}
		result, err := bridge.RunScript(bridge.Action{
			Action: "remove_song",
			File:   listPath,
			Path:   song.Filename,
		})
		if err == nil && result != nil && result.Status == "ok" {
			return deleteSongResultMsg{title: song.Title, filePath: song.FilePath}
		}
		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		} else if result != nil {
			errMsg = result.Error
		}
		return deleteSongResultMsg{title: song.Title, filePath: song.FilePath, errMsg: errMsg}
	}
}

func (m *HomeModel) renderDeleteOverlay(full string) string {
	songs := m.songs()
	songName := ""
	if m.deleteSongIdx >= 0 && m.deleteSongIdx < len(songs) {
		songName = songs[m.deleteSongIdx].Title
	}
	msg := ui.WhiteStyle.Render("  " + fmt.Sprintf(Tr("home.del_ask"), songName))
	purpleBtn := lipgloss.NewStyle().
		Background(lipgloss.Color("#BB86FC")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFFFFF")).
		Padding(0, 2)
	yesW := maxTrWidth("home.yes")
	noW := maxTrWidth("home.no")
	yesTxt := ui.FitPad(Tr("home.yes"), yesW)
	noTxt := ui.FitPad(Tr("home.no"), noW)
	noBtn := ui.ButtonStyle.Render(noTxt)
	yesBtn := ui.ErrorButtonStyle.Render(yesTxt)
	if m.deleteYes {
		yesBtn = purpleBtn.Render(yesTxt)
	} else {
		noBtn = purpleBtn.Render(noTxt)
	}
	btns := lipgloss.JoinHorizontal(lipgloss.Left, yesBtn, "  ", noBtn)
	hint := ui.DimStyle.Render(Tr("home.del_hint"))
	content := lipgloss.JoinVertical(lipgloss.Center, "", msg, "", btns, "", hint, "")
	content = ui.BorderStyle.Width(40).Render(ui.WhiteStyle.Bold(true).Render(Tr("home.del_title")) + "\n" + content)
	return m.placeOverlay(full, content)
}

func (m *HomeModel) startRename() {
	pl := state.Current.CurrentPlaylist
	if pl == nil {
		return
	}
	m.renameInput.SetValue(pl.Name)
	m.renameInput.SetCursor(len(pl.Name))
	m.renameInput.Focus()
	m.renameMode = true
}

func (m *HomeModel) handleRenameKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.renameMode = false
		m.renameInput.Blur()
		return m, nil
	case "enter":
		return m.saveRename()
	case "ctrl+a":
		if m.renameInput.Value() != "" {
			m.renameSelectAll = true
		}
		return m, nil
	default:
		if m.renameSelectAll {
			s := msg.String()
			if len(s) == 1 || s == "backspace" || s == "delete" {
				m.renameInput.SetValue("")
				m.renameInput.SetCursor(0)
				m.renameSelectAll = false
			} else {
				m.renameSelectAll = false
			}
		}
		var cmd tea.Cmd
		m.renameInput, cmd = m.renameInput.Update(msg)
		return m, cmd
	}
}

func (m *HomeModel) saveRename() (tea.Model, tea.Cmd) {
	m.renameMode = false
	m.renameInput.Blur()
	newName := strings.TrimSpace(m.renameInput.Value())
	if newName == "" {
		return m, nil
	}
	pl := state.Current.CurrentPlaylist
	cp := state.Current.CurrentProfile
	if pl == nil || cp == nil {
		return m, nil
	}
	if err := state.Current.SavePlaylistMeta(cp.FolderName, pl.FolderName, newName, pl.Bio); err != nil {
		m.addLog("error", err.Error())
		return m, nil
	}
	_ = state.Current.ScanProfiles()
	m.refreshAllContent()
	m.addLog("ok", "Playlist renamed: "+newName)
	return m, nil
}

func (m *HomeModel) playSelectedSong() tea.Cmd {
	songs := m.songs()
	if m.songFocusIdx >= 0 && m.songFocusIdx < len(songs) {
		return m.playSong(&songs[m.songFocusIdx])
	}
	return nil
}

// PlayResultMsg is returned after a play attempt
type PlayResultMsg struct {
	Title string
	Error error
}

// playSong starts a song picked by the user; it also ends a running shuffle
// session so the playlist continues linearly from that song.
func (m *HomeModel) playSong(song *state.Song) tea.Cmd {
	m.shuffleOrder = nil
	m.shufflePos = 0
	state.Current.Player.IsShuffled = false
	return m.startSong(song)
}

// startSong hands a song to the audio backend without touching the shuffle
// session — auto-advance uses it so the shuffled order survives playback.
func (m *HomeModel) startSong(song *state.Song) tea.Cmd {
	if song == nil {
		return nil
	}
	m.songEndedAt = time.Time{}
	m.manualStop = true
	// Stop any current playback before starting a new one
	if state.Current.Player.IsPlaying {
		bridge.PlayerCall(bridge.Action{Action: "stop"})
	}
	state.Current.Player.CurrentSong = song
	state.Current.Player.IsPlaying = true
	state.Current.Player.IsPaused = false
	state.Current.Player.StatusMsg = ""
	return func() tea.Msg {
		path := song.FilePath
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return PlayResultMsg{Title: song.Title, Error: fmt.Errorf("file not found: %s", path)}
		}
		pyPath := strings.ReplaceAll(path, "\\", "/")
		result, err := bridge.PlayerCall(bridge.Action{Action: "play", File: pyPath})
		if err == nil && result != nil {
			if result.Status == "error" {
				return PlayResultMsg{Title: song.Title, Error: fmt.Errorf("%s", result.Error)}
			}
			state.Current.Player.Duration = result.Duration
			state.Current.Player.Position = result.Position
			if result.Duration == 0 && song.Duration != "00:00" && song.Duration != "" {
				parts := strings.Split(song.Duration, ":")
				if len(parts) == 2 {
					m, s := 0, 0
					fmt.Sscanf(parts[0], "%d", &m)
					fmt.Sscanf(parts[1], "%d", &s)
					state.Current.Player.Duration = float64(m*60 + s)
				}
			}
		}
		return PlayResultMsg{Title: song.Title, Error: err}
	}
}

func (m *HomeModel) playAllSongs() tea.Cmd {
	pl := state.Current.CurrentPlaylist
	if pl == nil || len(pl.Songs) == 0 {
		return nil
	}
	return m.playSong(&pl.Songs[0])
}

func (m *HomeModel) playShuffledSongs() tea.Cmd {
	pl := state.Current.CurrentPlaylist
	if pl == nil || len(pl.Songs) == 0 {
		return nil
	}
	order := rand.Perm(len(pl.Songs))
	m.shuffleOrder = order
	m.shufflePos = 0
	state.Current.Player.IsShuffled = true
	return m.startSong(&pl.Songs[order[0]])
}

// nextSongTarget returns the track that should follow the current one.
// While a shuffle session runs it walks the shuffled order; when that order is
// exhausted (or the playlist changed under it) playback falls back to the
// linear order. The shuffle state is advanced here, synchronously with the
// model, so it can never be observed half-updated.
func (m *HomeModel) nextSongTarget() *state.Song {
	pl := state.Current.CurrentPlaylist
	if pl == nil || len(pl.Songs) == 0 {
		return nil
	}
	if state.Current.Player.IsShuffled && len(m.shuffleOrder) == len(pl.Songs) {
		if m.shufflePos < len(m.shuffleOrder)-1 {
			m.shufflePos++
			return &pl.Songs[m.shuffleOrder[m.shufflePos]]
		}
		state.Current.Player.IsShuffled = false
		m.shuffleOrder = nil
		m.shufflePos = 0
	}
	cur := state.Current.Player.CurrentSong
	if cur == nil {
		return &pl.Songs[0]
	}
	for i, s := range pl.Songs {
		if s.Filename == cur.Filename {
			return &pl.Songs[(i+1)%len(pl.Songs)]
		}
	}
	return &pl.Songs[0]
}

func (m *HomeModel) OnImportResult(msg ImportResultMsg) tea.Cmd {
	m.handleImportResult(msg)
	return nil
}

func (m *HomeModel) togglePlayPause() {
	ps := &state.Current.Player
	if ps.CurrentSong == nil {
		if state.Current.CurrentPlaylist != nil && len(state.Current.CurrentPlaylist.Songs) > 0 {
			m.playSong(&state.Current.CurrentPlaylist.Songs[0])
		}
		return
	}
	if ps.IsPlaying {
		ps.IsPaused = true
		ps.IsPlaying = false
		bridge.PlayerCall(bridge.Action{Action: "pause"})
	} else {
		ps.IsPlaying = true
		ps.IsPaused = false
		bridge.PlayerCall(bridge.Action{Action: "resume"})
	}
}

func (m *HomeModel) adjustVolume(delta float64) {
	v := state.Current.Player.Volume + delta
	// Respect the Sound → Volume limit: the app may only use up to that
	// percentage of the device's normal volume.
	maxV := float64(state.Current.SoundVolumeLimit) / 100.0
	if maxV <= 0 || maxV > 1 {
		maxV = 1
	}
	if v > maxV {
		v = maxV
	} else if v < 0 {
		v = 0
	}
	state.Current.Player.Volume = v
	go bridge.PlayerCall(bridge.Action{Action: "volume", Value: v})
}

func (m *HomeModel) processPlayerStatus(r *bridge.Result) {
	wasPlaying := state.Current.Player.IsPlaying
	state.Current.Player.Position = r.Position
	if r.Duration > 0 {
		state.Current.Player.Duration = r.Duration
	}
	state.Current.Player.AudioLevelL = r.AudioLevelL
	state.Current.Player.AudioLevelR = r.AudioLevelR
	if len(r.Spectrum) >= 17 {
		for i := 0; i < 17; i++ {
			v := r.Spectrum[i]
			if math.IsNaN(v) || v < 0 {
				v = 0
			}
			if v > 1 {
				v = 1
			}
			state.Current.Player.Spectrum[i] = v
			if v > m.smoothBands[i] {
				m.smoothBands[i] = m.smoothBands[i]*0.15 + v*0.85
			} else {
				m.smoothBands[i] = m.smoothBands[i]*0.3 + v*0.7
			}
		}
	}
	if r.Format != "" {
		state.Current.Player.Format = r.Format
		state.Current.Player.SampleRate = r.SampleRate
		state.Current.Player.Bitrate = r.Bitrate
	}
	switch r.Status {
	case "playing":
		state.Current.Player.IsPlaying = true
		state.Current.Player.IsPaused = false
		m.manualStop = false
	case "paused":
		state.Current.Player.IsPlaying = false
		state.Current.Player.IsPaused = true
	case "stopped", "idle":
		if wasPlaying {
			state.Current.Player.IsPlaying = false
			state.Current.Player.IsPaused = false
			if !m.manualStop {
				m.songEndedAt = time.Now()
			}
		}
		m.manualStop = false
	}

	// Animate global fade — rise on play, fall on pause/stop
	if r.Status == "playing" {
		if m.fadeLevel < 1 {
			m.fadeLevel += 0.5
			if m.fadeLevel > 1 {
				m.fadeLevel = 1
			}
		}
	} else {
		if m.fadeLevel > 0 {
			m.fadeLevel -= 0.5
			if m.fadeLevel < 0 {
				m.fadeLevel = 0
			}
		}
	}
}

func (m *HomeModel) checkAutoAdvance() tea.Cmd {
	if m.songEndedAt.IsZero() {
		return nil
	}
	if time.Since(m.songEndedAt) < 2*time.Second {
		return nil
	}
	m.songEndedAt = time.Time{}
	next := m.nextSongTarget()
	if next == nil {
		return nil
	}
	path := next.FilePath
	return func() tea.Msg {
		return PlaySongMsg{FilePath: path}
	}
}

func (m *HomeModel) selectPlaylist(idx int) {
	if state.Current.CurrentProfile != nil && idx < len(state.Current.CurrentProfile.Playlists) {
		state.Current.CurrentPlaylist = &state.Current.CurrentProfile.Playlists[idx]
		m.playlistIdx = idx
	}
}

func (m *HomeModel) refreshAllContent() {
	savedProfile := state.Current.CurrentProfile
	savedPlaylist := state.Current.CurrentPlaylist
	_ = state.Current.ScanProfiles()
	// Pick up files downloaded without registration (or copied manually).
	_ = bridge.ReconcileLibrary()
	_ = state.Current.ScanProfiles()
	if len(state.Current.Profiles) > 0 {
		if savedProfile != nil {
			found := false
			for i, p := range state.Current.Profiles {
				if p.FolderName == savedProfile.FolderName {
					state.Current.CurrentProfile = &state.Current.Profiles[i]
					found = true
					state.Current.CurrentPlaylist = nil
					if savedPlaylist != nil {
						for j, pl := range state.Current.CurrentProfile.Playlists {
							if pl.FolderName == savedPlaylist.FolderName {
								state.Current.CurrentPlaylist = &state.Current.CurrentProfile.Playlists[j]
								break
							}
						}
					}
					break
				}
			}
			if !found {
				state.Current.CurrentProfile = &state.Current.Profiles[0]
			}
		} else {
			state.Current.CurrentProfile = &state.Current.Profiles[0]
		}
		if state.Current.CurrentPlaylist == nil && len(state.Current.CurrentProfile.Playlists) > 0 {
			state.Current.CurrentPlaylist = &state.Current.CurrentProfile.Playlists[0]
		}
	}
	m.refreshPlaylistOptions()
	if state.Current.CurrentPlaylist != nil {
		for i, pl := range state.Current.CurrentProfile.Playlists {
			if pl.FolderName == state.Current.CurrentPlaylist.FolderName {
				m.playlistIdx = i
				break
			}
		}
	}
}

func (m *HomeModel) View() string {
	if m.width <= 0 {
		m.width = 120
	}
	if m.height <= 0 {
		m.height = 40
	}
	m.bodyHeight = m.height

	sidebar := m.viewSidebar(m.height)
	sidebarW := lipgloss.Width(sidebar)
	bodyW := m.width
	contentW := bodyW - sidebarW
	if contentW < 40 {
		contentW = 40
	}

	content := m.viewContent(m.height, contentW)
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, content)
	body = strings.TrimRight(body, "\n")

	if m.editModalOpen {
		return m.renderEditOverlay(body)
	}
	if m.deleteConfirm {
		return m.renderDeleteOverlay(body)
	}
	return body
}

func (m *HomeModel) viewSidebar(bodyH int) string {
	w := m.width / 3
	if w < 40 {
		w = 40
	}
	if w > 55 {
		w = 55
	}

	connectH := 6
	if bodyH < 14 {
		connectH = bodyH - 8
		if connectH < 5 {
			connectH = 5
		}
	}
	consoleH := (bodyH - connectH) * 6 / 10
	if consoleH < 10 {
		consoleH = 10
	}
	infoH := bodyH - connectH - consoleH
	if infoH < 6 {
		infoH = 6
		consoleH = bodyH - connectH - infoH
	}

	console := m.renderConsole(consoleH)
	cards := m.renderHomeConnectCards(w, connectH)
	info := m.renderInfoPanel(infoH)
	return lipgloss.JoinVertical(lipgloss.Left, console, cards, info)
}

func (m *HomeModel) renderHomeConnectCards(w, h int) string {
	if h < 6 {
		return strings.Repeat(" ", w)
	}
	cardW := (w - 3) / 2
	if cardW < 12 {
		cardW = 12
	}
	cardH := (h - 2) / 3
	if cardH < 3 {
		cardH = 3
	}
	spot := m.renderHomeCard("Spotify",
		ui.ColorSpotifyLight, ui.ColorSpotifyFocus, m.connectFocus == 0 && m.sidebarFocus == 1, cardW, cardH)
	yt := m.renderHomeCard("YouTube Music",
		ui.ColorYouTubeLight, ui.ColorYouTubeFocus, m.connectFocus == 1 && m.sidebarFocus == 1, cardW, cardH)
	return lipgloss.JoinHorizontal(lipgloss.Top, spot, " ", yt)
}

func (m *HomeModel) renderHomeCard(name string, base, focus lipgloss.Color, focused bool, w, h int) string {
	// On focus the border and label adopt the active theme color.
	border := base
	labelColor := base
	if focused {
		border = ui.ColorAccent
		labelColor = ui.ColorAccent
	}
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(w).
		Height(h).
		Align(lipgloss.Center, lipgloss.Center)
	if focused {
		style = style.Bold(true)
	}
	label := lipgloss.NewStyle().Foreground(labelColor).Bold(true).Render(name)
	return style.Render(label)
}

func (m *HomeModel) launchConnectFromHome() (tea.Model, tea.Cmd) {
	plat := browser.PlatformSpotify
	if m.connectFocus == 1 {
		plat = browser.PlatformYouTube
	}
	return m, func() tea.Msg { return homeConnectMsg{platform: plat} }
}

func (m *HomeModel) addLog(level, msg string) {
	now := time.Now().Format("15:04:05")
	var line string
	switch level {
	case "error":
		line = fmt.Sprintf("x %s %s", now, msg)
	case "ok":
		line = fmt.Sprintf("v %s %s", now, msg)
	default:
		line = fmt.Sprintf("> %s %s", now, msg)
	}
	m.logLines = append(m.logLines, line)
	if len(m.logLines) > 200 {
		m.logLines = m.logLines[len(m.logLines)-200:]
	}
}

func (m *HomeModel) renderConsole(bodyH int) string {
	w := 38
	if m.width > 0 {
		w = m.width / 3
		if w < 40 {
			w = 40
		}
		if w > 55 {
			w = 55
		}
	}
	title := ui.SectionTitleStyle.Render(langT("CONSOLE", "KONSOL"))
	visible := bodyH - 4
	if visible < 8 {
		visible = 8
	}
	contentW := w - 4

	logCount := len(m.logLines)
	totalLines := logCount

	maxScroll := totalLines - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.consoleScroll < 0 || m.consoleScroll > maxScroll {
		m.consoleScroll = maxScroll
	}

	start := m.consoleScroll
	end := start + visible
	if end > totalLines {
		end = totalLines
	}

	haveScrollbar := totalLines > visible

	var inner string
	if totalLines == 0 {
		inner = title + "\n" + ui.FaintStyle.Render("  "+Tr("dl.no_logs"))
	} else {
		var contentParts []string
		for i := start; i < end; i++ {
			raw := m.logLines[i]
			if len(raw) > contentW-1 {
				raw = raw[:contentW-1]
			}
			if strings.HasPrefix(raw, "x ") {
				contentParts = append(contentParts, ui.ErrorStyle.Render(raw))
			} else if strings.HasPrefix(raw, "v ") {
				contentParts = append(contentParts, lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render(raw))
			} else {
				contentParts = append(contentParts, ui.FaintStyle.Render(raw))
			}
		}
		contentStr := strings.Join(contentParts, "\n")

		if haveScrollbar {
			thumbPos := 0
			if maxScroll > 0 {
				thumbPos = int(float64(m.consoleScroll) / float64(maxScroll) * float64(visible-1))
			}
			var scrollParts []string
			for i := 0; i < visible; i++ {
				if i == thumbPos {
					scrollParts = append(scrollParts, ui.AccentStyle.Render("█"))
				} else {
					scrollParts = append(scrollParts, ui.FaintStyle.Render("│"))
				}
			}
			scrollStr := strings.Join(scrollParts, "\n")
			contentStr = lipgloss.JoinHorizontal(lipgloss.Top, contentStr, " ", scrollStr)
		}

		inner = title + "\n" + contentStr
	}

	innerH := lipgloss.Height(inner)
	targetH := bodyH - 2
	if innerH < targetH {
		inner += strings.Repeat("\n", targetH-innerH)
	}

	consoleStyle := ui.BorderStyle
	if m.sidebarFocus == 0 && m.sectionFocus == 0 {
		consoleStyle = ui.AccentBorderStyle
	}
	box := consoleStyle.Width(w).Render(inner)
	boxH := lipgloss.Height(box)
	if boxH < bodyH {
		box += strings.Repeat("\n", bodyH-boxH)
	}
	return box
}

func (m *HomeModel) renderInfoPanel(bodyH int) string {
	w := 38
	if m.width > 0 {
		w = m.width / 3
		if w < 40 {
			w = 40
		}
		if w > 55 {
			w = 55
		}
	}
	innerW := w - 2
	if innerW < 30 {
		innerW = 30
	}

	title := ui.SectionTitleStyle.Render(langT("SPECTRUM", "SPEKTRUM"))
	spectrumRows := bodyH - 4
	if spectrumRows < 2 {
		spectrumRows = 2
	}
	spectrum := m.renderSpectrum(spectrumRows, innerW)

	inner := lipgloss.JoinVertical(lipgloss.Left, title, "", spectrum)
	innerH := lipgloss.Height(inner)
	targetH := bodyH - 2
	if innerH < targetH {
		inner += strings.Repeat("\n", targetH-innerH)
	}

	sectionStyle := ui.BorderStyle
	return sectionStyle.Width(w).Render(inner)
}

func (m *HomeModel) renderSpectrum(rows, w int) string {
	if rows < 1 || w < 30 {
		return ""
	}

	var bands [17]float64
	for i := 0; i < 17; i++ {
		bands[i] = m.smoothBands[i] * m.fadeLevel
	}
	barW := 2
	gap := 1
	totalBars := 17
	totalW := totalBars*barW + (totalBars-1)*gap
	if totalW > w {
		barW = 1
		totalW = totalBars*barW + (totalBars-1)*gap
	}
	pad := (w - totalW) / 2
	if pad < 0 {
		pad = 0
	}
	rightPad := w - totalW - pad
	if rightPad < 0 {
		rightPad = 0
	}

	var sb strings.Builder
	for row := rows - 1; row >= 0; row-- {
		sb.WriteString(strings.Repeat(" ", pad))
		for b := 0; b < totalBars; b++ {
			level := bands[b]
			rowBottom := float64(row) / float64(rows)
			rowTop := float64(row+1) / float64(rows)
			col := ui.SpectrumColor(b, level)
			var ch string
			if level >= rowTop {
				ch = "█"
			} else if level > rowBottom {
				frac := (level - rowBottom) / (rowTop - rowBottom)
				idx := int(frac * 7)
				if idx > 7 {
					idx = 7
				}
				ch = []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}[idx]
			} else {
				ch = " "
			}
			if barW > 1 {
				sb.WriteString(col.Render(ch + ch))
			} else {
				sb.WriteString(col.Render(ch))
			}
			if b < totalBars-1 {
				sb.WriteString(strings.Repeat(" ", gap))
			}
		}
		sb.WriteString(strings.Repeat(" ", rightPad))
		if row > 0 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *HomeModel) viewContent(bodyH, contentW int) string {
	plInfo := m.viewPlaylistInfo(bodyH)
	songsW := contentW - 42
	if songsW < 20 {
		songsW = 20
	}
	tableW := songsW
	tableTitle := ui.WhiteStyle.Bold(true).Render(" " + langT("SONGS", "SARKILAR") + " ")
	if m.sectionFocus == 2 || m.focusIdx == 6 {
		tableTitle = ui.AccentStyle.Bold(true).Render(" " + langT("SONGS", "SARKILAR") + " ")
	}
	hint := ui.DimStyle.Render(Tr("home.songs_hint"))
	borderStyle := ui.BorderStyle
	if m.sectionFocus == 2 || m.focusIdx == 6 {
		borderStyle = ui.AccentBorderStyle
	}
	maxVis := m.maxVisibleSongs()
	if maxVis < 1 {
		maxVis = 1
	}
	if m.songOffset < 0 {
		m.songOffset = 0
	}
	songs := m.songs()
	if m.songOffset > len(songs)-maxVis && m.songOffset > 0 {
		m.songOffset = len(songs) - maxVis
	}
	songsHTML := m.renderSongs(tableW-4, m.songOffset, maxVis)

	songsH := lipgloss.Height(songsHTML)
	availH := bodyH - 4
	padding := availH - songsH
	if padding < 0 {
		padding = 0
	}
	songsHTML += strings.Repeat("\n", padding)

	tableBox := borderStyle.Width(tableW).Render(tableTitle + "\n" + songsHTML + "\n" + hint)
	return lipgloss.JoinHorizontal(lipgloss.Top, plInfo, tableBox)
}

func (m *HomeModel) renderSongs(w, offset, max int) string {
	songs := m.songs()
	if len(songs) == 0 {
		return ui.DimStyle.Render("  " + Tr("home.no_songs"))
	}

	if offset >= len(songs) {
		offset = 0
	}
	end := offset + max
	if end > len(songs) {
		end = len(songs)
	}

	selectedBg := ui.ColorSelection
	titleStyle := ui.WhiteStyle.Bold(true)
	artistStyle := ui.DimStyle
	headerStyle := ui.DimStyle.Bold(true)
	btnActiveText := ui.WhiteStyle.Bold(true)
	btnInactiveText := ui.DimStyle

	isFocused := m.focusIdx == 6

	numW := 4
	durW := 9
	// Row action buttons keep the width of their longest translation across
	// all 11 languages; the actions column grows to fit them.
	playW := maxTrWidth("home.act_play")
	editW := maxTrWidth("home.act_edit")
	delW := maxTrWidth("home.act_del")
	actionsW := playW + editW + delW + 2
	if actionsW < 17 {
		actionsW = 17
	}
	narrowActions := false
	if w < 55 {
		actionsW = 13
		narrowActions = true
	}
	// Format: " %s %s %s %s %s " (6 spaces + columns)
	// Overhead = 6 + numW(4) + durW(9) = 19. Border adds 2. Content must be w-2.
	// 19 + songW + actionsW = w-2 → songW = w-21-actionsW
	songW := w - 21 - actionsW
	if songW < 8 {
		songW = 8
	}
	titleW := songW * 3 / 5
	artistW := songW - titleW

	numCol := lipgloss.NewStyle().MaxWidth(numW).Width(numW).Align(lipgloss.Center)
	titleCol := lipgloss.NewStyle().MaxWidth(titleW).Width(titleW).Align(lipgloss.Left)
	artistCol := lipgloss.NewStyle().MaxWidth(artistW).Width(artistW).Align(lipgloss.Center)
	durCol := lipgloss.NewStyle().MaxWidth(durW).Width(durW).Align(lipgloss.Center)
	actCol := lipgloss.NewStyle().MaxWidth(actionsW).Width(actionsW).Align(lipgloss.Center)

	hNum := numCol.Render("#")
	hTitle := titleCol.Render(langT("Title", "Isim"))
	hArtist := artistCol.Render(langT("Artist", "Sanatci"))
	hDur := durCol.Render(langT("Dur.", "Sre."))
	hAct := actCol.Render(langT("Operations", "Islemler"))
	h := headerStyle.Render(fmt.Sprintf(" %s %s %s %s %s ", hNum, hTitle, hArtist, hDur, hAct))
	headerBorder := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorBorder).
		Width(w)
	items := []string{headerBorder.Render(h)}

	for i := offset; i < end; i++ {
		song := songs[i]
		title := song.Title
		artist := song.Artist

		if len(title) > titleW {
			truncLen := titleW - 3
			if truncLen < 1 {
				truncLen = 1
			}
			title = title[:truncLen] + "..."
		}
		if len(artist) > artistW {
			truncLen := artistW - 3
			if truncLen < 1 {
				truncLen = 1
			}
			artist = artist[:truncLen] + "..."
		}

		numStr := numCol.Render(fmt.Sprintf("%d.", i+1))
		titleR := titleCol.Render(titleStyle.Render(title))
		artistR := artistCol.Render(artistStyle.Render(artist))
		dur := durCol.Render(normalizeDuration(song.Duration))

		isThisFocused := isFocused && m.songFocusIdx == i
		af := m.songActionFocus

		var playBtn, editBtn, delBtn string
		actFmt := "%-5s%-5s%-4s"
		if !narrowActions {
			playBtn = ui.FitPad(Tr("home.act_play"), playW)
			editBtn = ui.FitPad(Tr("home.act_edit"), editW)
			delBtn = ui.FitPad(Tr("home.act_del"), delW)
			actFmt = fmt.Sprintf("%%-%ds%%-%ds%%-%ds", playW, editW, delW)
		} else {
			playBtn = " P"
			editBtn = " E"
			delBtn = " D"
		}

		if isThisFocused {
			switch af {
			case 0:
				playBtn = btnActiveText.Render(playBtn)
			case 1:
				editBtn = btnActiveText.Render(editBtn)
			case 2:
				delBtn = btnActiveText.Render(delBtn)
			}
		} else {
			playBtn = btnInactiveText.Render(playBtn)
			editBtn = btnInactiveText.Render(editBtn)
			delBtn = btnInactiveText.Render(delBtn)
		}

		line := fmt.Sprintf(" %s %s %s %s %s ", numStr, titleR, artistR, dur, actCol.Render(fmt.Sprintf(actFmt, playBtn, editBtn, delBtn)))

		if m.focusIdx == 6 && m.songFocusIdx == i {
			songStyle := ui.AccentBorderStyle.
				Width(w).
				Background(selectedBg)
			items = append(items, songStyle.Render(line))
		} else {
			songStyle := ui.BorderStyle.
				Width(w)
			items = append(items, songStyle.Render(line))
		}
	}
	return strings.Join(items, "\n")
}

func (m *HomeModel) viewPlaylistInfo(bodyH int) string {
	pl := state.Current.CurrentPlaylist
	cp := state.Current.CurrentProfile
	border := ui.BorderStyle
	if m.sectionFocus == 1 {
		border = ui.AccentBorderStyle
	}
	titleStyle := ui.WhiteStyle
	if m.sectionFocus == 1 {
		titleStyle = ui.AccentStyle
	}
	title := titleStyle.Bold(true).Render(" " + langT("PLAYLIST", "PLAYLIST") + " ")
	if pl == nil || cp == nil {
		pad := bodyH - 6
		if pad < 0 {
			pad = 0
		}
		inner := title + "\n" + ui.DimStyle.Render("\n  "+Tr("common.no_playlist")) + strings.Repeat("\n", pad)
		return border.Width(38).Render(inner)
	}

	// Use preview index for name when browsing
	displayPl := pl
	if m.sectionFocus == 1 && m.playlistActionFocus == 0 && m.previewIdx >= 0 && m.previewIdx < len(m.playlistOptions) {
		if cp != nil && m.previewIdx < len(cp.Playlists) {
			displayPl = &cp.Playlists[m.previewIdx]
		}
	}
	plName := displayPl.Name
	if len(plName) > 36 {
		plName = plName[:33] + "..."
	}
	plBio := displayPl.Bio
	if len(plBio) > 36 {
		plBio = plBio[:33] + "..."
	}
	nameStyle := ui.WhiteStyle.Bold(true)
	if m.sectionFocus == 1 && m.playlistActionFocus == 0 {
		nameStyle = ui.AccentStyle.Bold(true)
	}
	name := nameStyle.Render("  " + plName)
	bio := ui.DimStyle.Render("  " + plBio)

	// Art section (centered, always reserves space): shows the cover of
	// the currently playing song, decoded from its embedded artwork.
	var artStr string
	baseH := 12
	targetH := bodyH - 3
	avail := targetH - baseH
	artRows := 0
	if avail >= 4 {
		artRows = 18
		if avail < artRows {
			artRows = avail
		}
		if img := m.nowPlayingCover(); img != nil {
			artStr = renderCoverArt(img, 36, artRows)
		}
	}

	// Created date (centered in 36-col content area)
	cw := 36
	created := ""
	if displayPl.CreatedAt != "" {
		txt := Tr("home.created") + displayPl.CreatedAt
		created = ui.DimStyle.Render(padCenter(txt, cw))
	}

	// Duration + song count (side by side, centered)
	totalSecs := 0
	for _, s := range displayPl.Songs {
		totalSecs += parseDuration(s.Duration)
	}
	durStr := formatDuration(totalSecs)
	infoTxt := fmt.Sprintf("%s    "+Tr("home.songs_count"), durStr, len(displayPl.Songs))
	infoLine := ui.AccentStyle.Render(padCenter(infoTxt, cw))

	// Action buttons (centered). Both keep the width of their longest
	// translation across all 11 languages, so language switches never
	// move them inside the 36-column card.
	playTxt := ui.FitPad(Tr("home.play_all"), maxTrWidth("home.play_all"))
	shufTxt := ui.FitPad(Tr("home.shuffle"), maxTrWidth("home.shuffle"))
	playFocused := m.sectionFocus == 1 && m.playlistActionFocus == 1
	shufFocused := m.sectionFocus == 1 && m.playlistActionFocus == 2
	playBtn := ui.DimStyle.Render(playTxt)
	shufBtn := ui.DimStyle.Render(shufTxt)
	if playFocused {
		playBtn = ui.AccentStyle.Render(playTxt)
	}
	if shufFocused {
		shufBtn = ui.AccentStyle.Render(shufTxt)
	}
	btnTxt := lipgloss.JoinHorizontal(lipgloss.Center, playBtn, "     ", shufBtn)
	btnLine := padCenter(btnTxt, cw)

	inner := lipgloss.JoinVertical(lipgloss.Left, "", name, bio)
	if artRows > 0 {
		if artStr != "" {
			artLines := strings.Split(artStr, "\n")
			for j, line := range artLines {
				artLines[j] = strings.Repeat(" ", 1) + line
			}
			inner += "\n" + strings.Join(artLines, "\n")
		} else {
			inner += strings.Repeat("\n", artRows)
		}
	}
	inner += "\n\n" + created
	inner += "\n\n" + infoLine
	inner += "\n\n" + btnLine

	innerH := lipgloss.Height(inner)
	if innerH < targetH {
		inner += strings.Repeat("\n", targetH-innerH)
	}
	return border.Width(38).Render(title + "\n" + inner)
}

func parseDuration(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	if len(parts) == 2 {
		var m, sec int
		fmt.Sscanf(parts[0], "%d", &m)
		fmt.Sscanf(parts[1], "%d", &sec)
		return m*60 + sec
	}
	if len(parts) == 1 {
		var sec int
		fmt.Sscanf(parts[0], "%d", &sec)
		return sec
	}
	return 0
}

// normalizeDuration ensures a song duration is rendered as mm:ss. Values that
// are already in mm:ss form are passed through; bare integers (raw seconds,
// e.g. legacy song_list.txt entries) are converted so the list always shows
// dakika:saniye.
func normalizeDuration(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "00:00"
	}
	if strings.Contains(s, ":") {
		return s
	}
	secs, err := strconv.Atoi(s)
	if err != nil {
		return s
	}
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}

func formatDuration(totalSecs int) string {
	h := totalSecs / 3600
	m := (totalSecs % 3600) / 60
	if h > 0 {
		return fmt.Sprintf(Tr("home.dur_h"), h, m)
	}
	return fmt.Sprintf(Tr("home.dur_m"), m)
}

func padCenter(s string, w int) string {
	vw := lipgloss.Width(s)
	if vw >= w {
		return s
	}
	l := (w - vw) / 2
	return strings.Repeat(" ", l) + s
}

// nowPlayingCover returns the decoded cover art of the currently playing
// song, read from its embedded picture and cached per file path. It returns
// nil when nothing plays or the file carries no artwork.
func (m *HomeModel) nowPlayingCover() image.Image {
	song := state.Current.Player.CurrentSong
	if song == nil || song.FilePath == "" {
		m.coverPath = ""
		m.coverImg = nil
		return nil
	}
	if song.FilePath == m.coverPath {
		return m.coverImg
	}
	m.coverPath = song.FilePath
	m.coverImg = nil
	f, err := os.Open(song.FilePath)
	if err != nil {
		return nil
	}
	defer f.Close()
	meta, err := tag.ReadFrom(f)
	if err != nil || meta.Picture() == nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(meta.Picture().Data))
	if err != nil {
		return nil
	}
	m.coverImg = img
	return img
}

func renderCoverArt(img image.Image, cols, rows int) string {
	if img == nil {
		return ""
	}
	// Inner area (excluding border)
	inCols := cols - 2
	inRows := rows - 2
	if inCols < 4 || inRows < 2 {
		return ""
	}

	// Resize: each cell shows 2 vertical pixels via half-block (▄)
	pixelW := inCols
	pixelH := inRows * 2
	resized := scaleImage(img, pixelW, pixelH)
	applyRoundedCorners(resized, pixelW/12)

	// Extract dominant color from resized image
	dr, dg, db := averageColor(resized)

	var out strings.Builder
	domFg := fmt.Sprintf("\033[38;2;%d;%d;%dm", dr, dg, db)
	reset := "\033[0m"

	// Top border line
	out.WriteString(domFg + "┌" + strings.Repeat("─", inCols) + "┐" + reset + "\n")

	// Image rows with side borders
	for cy := 0; cy < inRows; cy++ {
		out.WriteString(domFg + "│" + reset)
		for cx := 0; cx < inCols; cx++ {
			r1, g1, b1, a1 := resized.At(cx, cy*2).RGBA()
			r2, g2, b2, a2 := resized.At(cx, cy*2+1).RGBA()

			if a1 < 128 && a2 < 128 {
				out.WriteByte(' ')
				continue
			}
			if a2 < 128 {
				out.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▀\033[0m", r1>>8, g1>>8, b1>>8))
				continue
			}
			if a1 < 128 {
				out.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%dm▄\033[0m", r2>>8, g2>>8, b2>>8))
			} else {
				out.WriteString(fmt.Sprintf("\033[38;2;%d;%d;%d;48;2;%d;%d;%dm▄\033[0m", r2>>8, g2>>8, b2>>8, r1>>8, g1>>8, b1>>8))
			}
		}
		out.WriteString(domFg + "│" + reset + "\n")
	}

	// Bottom border line
	out.WriteString(domFg + "└" + strings.Repeat("─", inCols) + "┘" + reset)

	return out.String()
}

func scaleImage(img image.Image, dstW, dstH int) *image.RGBA {
	src := img.Bounds()
	srcW := src.Dx()
	srcH := src.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	for y := 0; y < dstH; y++ {
		for x := 0; x < dstW; x++ {
			sx := x * srcW / dstW
			sy := y * srcH / dstH
			dst.Set(x, y, img.At(sx, sy))
		}
	}
	return dst
}

func applyRoundedCorners(img *image.RGBA, r int) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// Top-left
			if x < r && y < r {
				dx, dy := r-x, r-y
				if dx*dx+dy*dy > r*r {
					img.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
				}
			}
			// Top-right
			if x >= w-r && y < r {
				dx, dy := x-(w-r-1), r-y
				if dx*dx+dy*dy > r*r {
					img.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
				}
			}
			// Bottom-left
			if x < r && y >= h-r {
				dx, dy := r-x, y-(h-r-1)
				if dx*dx+dy*dy > r*r {
					img.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
				}
			}
			// Bottom-right
			if x >= w-r && y >= h-r {
				dx, dy := x-(w-r-1), y-(h-r-1)
				if dx*dx+dy*dy > r*r {
					img.SetRGBA(x, y, color.RGBA{0, 0, 0, 0})
				}
			}
		}
	}
}

func averageColor(img *image.RGBA) (uint8, uint8, uint8) {
	b := img.Bounds()
	var tr, tg, tb uint64
	var n uint64
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a > 128 {
				tr += uint64(r >> 8)
				tg += uint64(g >> 8)
				tb += uint64(bl >> 8)
				n++
			}
		}
	}
	if n == 0 {
		return 100, 100, 100
	}
	return uint8(tr / n), uint8(tg / n), uint8(tb / n)
}
