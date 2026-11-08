package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MusicLe/bridge"
	"MusicLe/internal/browser"
	"MusicLe/state"
	"MusicLe/ui"
)

// ConnectModel is the platform-selection screen of the browser connector.
// It shows two brand-colored cards (Spotify / YouTube Music) between which the
// user navigates with F1 or the arrow keys. After a choice it runs a browser
// scan behind a loading modal and then lists the discovered playlists.
type ConnectModel struct {
	width  int
	height int
	focus  int // 0 = Spotify, 1 = YouTube Music
	chosen browser.Platform

	scanning  bool
	scanStart time.Time
	scanDone  bool
	playlists []browser.Playlist
	scanErr   error

	plFocus   int
	confirmed map[int]bool
	saved     bool
	saveErr   error
}

func NewConnectModel() *ConnectModel {
	return &ConnectModel{focus: 0}
}

func (m *ConnectModel) Init() tea.Cmd { return nil }

func (m *ConnectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch {
		case m.scanning:
			return m, nil

		case m.scanDone && m.scanErr != nil:
			switch k.String() {
			case "esc", "enter", " ":
				m.scanning = false
				m.scanDone = false
				m.scanErr = nil
				return m, nil
			}
			return m, nil

		case m.scanDone && m.scanErr == nil:
			switch k.String() {
			case "up", "k":
				if m.plFocus > 0 {
					m.plFocus--
				}
				return m, nil
			case "down", "j":
				if m.plFocus < len(m.playlists)-1 {
					m.plFocus++
				}
				return m, nil
			case "enter", " ":
				if !m.confirmed[m.plFocus] {
					if err := m.confirm(m.plFocus); err != nil {
						m.saveErr = err
					}
				}
				if m.saved {
					return m, func() tea.Msg { return connectDoneMsg{} }
				}
				return m, nil
			case "esc":
				if m.saved {
					return m, func() tea.Msg { return connectDoneMsg{} }
				}
				m.scanDone = false
				m.playlists = nil
				m.plFocus = 0
				m.confirmed = nil
				return m, nil
			}

		default:
			switch k.String() {
			case "left", "up", "shift+tab":
				m.focus = 0
				return m, nil
			case "right", "down", "tab":
				m.focus = 1
				return m, nil
			case "enter", " ":
				if m.focus == 0 {
					m.chosen = browser.PlatformSpotify
				} else {
					m.chosen = browser.PlatformYouTube
				}
				m.scanning = true
				m.scanStart = time.Now()
				return m, scanConnectCmd(m.chosen)
			}
		}
	}
	return m, nil
}

// confirm saves the playlist at index i: it derives an auto-named profile for
// the chosen platform and imports the tracks via bridge.ImportFromBrowser.
func (m *ConnectModel) confirm(i int) error {
	if m.confirmed == nil {
		m.confirmed = map[int]bool{}
	}
	if m.confirmed[i] {
		return nil
	}
	pl := m.playlists[i]
	displayName, folderName := browser.NextProfileName(state.Current.Profiles, m.chosen, state.Current.ProfilesDir())
	if err := state.Current.CreateProfileStructure(folderName, displayName, "", state.Current.Language); err != nil {
		return err
	}
	plFolder := uniqueFolder(browser.Slugify(pl.Name), folderName)
	if err := bridge.ImportFromBrowser(folderName, plFolder, pl.Name, pl.Tracks); err != nil {
		return err
	}
	if err := state.Current.ScanProfiles(); err != nil {
		return err
	}
	for idx := range state.Current.Profiles {
		if state.Current.Profiles[idx].FolderName == folderName {
			state.Current.CurrentProfile = &state.Current.Profiles[idx]
			if len(state.Current.CurrentProfile.Playlists) > 0 {
				state.Current.CurrentPlaylist = &state.Current.CurrentProfile.Playlists[0]
			}
			break
		}
	}
	m.confirmed[i] = true
	m.saved = true
	return nil
}

// uniqueFolder returns a playlist folder name that does not yet exist under the
// given profile directory.
func uniqueFolder(name, profileFolder string) string {
	base := name
	i := 2
	for {
		if _, err := os.Stat(state.Current.PlaylistDir(profileFolder, name)); os.IsNotExist(err) {
			break
		}
		name = fmt.Sprintf("%s_%d", base, i)
		i++
	}
	return name
}

// finishScan stores the result of a browser scan and dismisses the modal.
func (m *ConnectModel) finishScan(pls []browser.Playlist, err error) {
	m.scanning = false
	m.scanDone = true
	m.playlists = pls
	m.scanErr = err
}

// scanConnectCmd runs the real CDP scan but guarantees the loading modal is
// visible for at least 2 seconds (simulated API latency).
func scanConnectCmd(platform browser.Platform) tea.Cmd {
	return func() tea.Msg {
		start := time.Now()
		pls, err := browser.Connect(platform)
		if d := time.Since(start); d < 2*time.Second {
			time.Sleep(2*time.Second - d)
		}
		return ConnectResultMsg{Platform: platform, Playlists: pls, Err: err}
	}
}

func (m *ConnectModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	// The cards (base) stay visible behind the modal so the screen is never a
	// flat black void — the error/loading dialog is overlaid on top of them.
	if m.scanning {
		return placeOverlay(m.renderBase(), m.renderModal(), m.width)
	}
	if m.scanDone {
		if m.scanErr != nil {
			return placeOverlay(m.renderBase(), m.renderErrorModal(m.scanErr), m.width)
		}
		return m.renderPlaylistList()
	}

	return m.renderBase()
}

// renderBase draws the platform selection cards, filling the whole screen so a
// modal can be overlaid centered on top of it.
func (m *ConnectModel) renderBase() string {
	title := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).
		Render(Tr("connect.title"))

	cardW := (m.width - 10) / 2
	if cardW < 24 {
		cardW = 24
	}
	cardH := (m.height - 16) / 4
	if cardH < 4 {
		cardH = 4
	}

	spotify := m.renderCard("Spotify", ui.ColorSpotify, ui.ColorSpotifyFocus, m.focus == 0, cardW, cardH)
	yt := m.renderCard("YouTube Music", ui.ColorYouTube, ui.ColorYouTubeFocus, m.focus == 1, cardW, cardH)

	row := lipgloss.JoinHorizontal(lipgloss.Top, spotify, "    ", yt)

	footer := lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render(
		Tr("connect.footer"))

	content := lipgloss.JoinVertical(lipgloss.Center, title, "", row, "", footer)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

// renderErrorModal shows a scan failure as a small, dismissible modal (Esc).
func (m *ConnectModel) renderErrorModal(err error) string {
	innerW := 56
	// Small close square in the top-right corner: theme-colored background with
	// a black X mark.
	closeBtn := lipgloss.NewStyle().
		Background(ui.ColorAccent).
		Foreground(ui.ColorBlack).
		Bold(true).
		Padding(0, 1).
		Render("✕")
	header := lipgloss.PlaceHorizontal(innerW, lipgloss.Right, closeBtn)

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorError).
		Padding(1, 4).
		Align(lipgloss.Center)
	content := lipgloss.JoinVertical(lipgloss.Center,
		header,
		"",
		lipgloss.NewStyle().Foreground(ui.ColorError).Bold(true).Render(Tr("connect.conn_error")),
		"",
		lipgloss.NewStyle().Foreground(ui.ColorSecondary).Width(innerW).Align(lipgloss.Center).Render(err.Error()),
		"",
		lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render(Tr("connect.esc_close")),
	)
	return box.Render(content)
}

// renderModal shows the "reading browser info" loader while scanning.
func (m *ConnectModel) renderModal() string {
	frames := []string{"|", "/", "-", "\\"}
	spin := frames[(time.Now().Nanosecond()/100000000)%4]
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Padding(1, 4).
		Align(lipgloss.Center)
	content := lipgloss.JoinVertical(lipgloss.Center,
		lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).Render(Tr("connect.fetching")),
		"",
		lipgloss.NewStyle().Foreground(ui.ColorAccent).Render(spin+" "+string(m.chosen)+" "+Tr("connect.scanning")),
	)
	return box.Render(content)
}

// renderPlaylistList shows the discovered playlists, each with an "Onayla"
// button. The focused row is confirmed with Enter, which advances to the next
// unconfirmed row so playlists can be approved sequentially.
func (m *ConnectModel) renderPlaylistList() string {
	header := lipgloss.NewStyle().Foreground(ui.ColorPrimary).Bold(true).
		Render(fmt.Sprintf(Tr("connect.found"), m.chosen, len(m.playlists)))

	var rows []string
	for i, pl := range m.playlists {
		cursor := "  "
		if i == m.plFocus {
			cursor = "▶ "
		}
		name := cursor + pl.Name
		count := lipgloss.NewStyle().Foreground(ui.ColorSecondary).
			Render(fmt.Sprintf(Tr("connect.songs_count"), pl.TrackCount))

		// Both states share one fixed width (longest translation of either
		// label across all 11 languages) so rows never shift on confirm.
		actW := maxTrWidth("connect.confirm")
		if w := maxTrWidth("connect.confirmed"); w > actW {
			actW = w
		}
		confirmTxt := ui.FitPad(Tr("connect.confirm"), actW)
		confirmedTxt := ui.FitPad(Tr("connect.confirmed"), actW)
		var action string
		switch {
		case m.confirmed[i]:
			action = lipgloss.NewStyle().Foreground(ui.ColorSuccess).Render(confirmedTxt)
		case i == m.plFocus:
			action = ui.AccentButtonStyle.Render(confirmTxt)
		default:
			action = ui.ButtonStyle.Render(confirmTxt)
		}
		row := lipgloss.JoinHorizontal(lipgloss.Center, name, "   ", count, "   ", action)
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = append(rows, lipgloss.NewStyle().Foreground(ui.ColorSecondary).Render(Tr("connect.no_playlist_found")))
	}

	body := strings.Join(rows, "\n")

	var footer string
	switch {
	case m.saveErr != nil:
		footer = lipgloss.NewStyle().Foreground(ui.ColorError).
			Render(fmt.Sprintf(Tr("connect.save_error"), m.saveErr))
	case m.saved:
		footer = lipgloss.NewStyle().Foreground(ui.ColorSuccess).
			Render(Tr("connect.saved"))
	default:
		footer = lipgloss.NewStyle().Foreground(ui.ColorSecondary).
			Render(Tr("connect.nav_hint"))
	}

	return lipgloss.JoinVertical(lipgloss.Center, header, "", body, "", footer)
}

func (m *ConnectModel) renderCard(name string, base, focus lipgloss.Color, focused bool, w, h int) string {
	// Border + label adopt the active theme color when the card is focused.
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
