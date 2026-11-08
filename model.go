package main

import (
	"net"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MusicLe/bridge"
	"MusicLe/components"
	"MusicLe/internal/browser"
	"MusicLe/state"
	"MusicLe/ui"
)

type StartDownloadMsg struct {
	Action string
	URL    string
	Output string
}

type DownloadResultMsg struct {
	Result *bridge.Result
	Error  error
	Action string
}

type LocalFileImportMsg struct {
	FilePath string
	Output   string
}

type ImportResultMsg struct {
	Result *bridge.Result
	Error  error
}

type PlaySongMsg struct {
	FilePath string
}

// deleteSongResultMsg reports the outcome of an async song deletion back to
// the home model, which then refreshes the library and logs the result.
type deleteSongResultMsg struct {
	title    string
	filePath string
	errMsg   string
}

type PlayerCmdMsg struct {
	Action string
	Value  float64
}

type ThemeChangedMsg struct{}

type setupDoneMsg struct{}
type errorMsg string

// ConnectResultMsg carries the result of a browser scan back to the UI.
type ConnectResultMsg struct {
	Platform  browser.Platform
	Playlists []browser.Playlist
	Err       error
}

// connectDoneMsg returns the UI to the home screen after a successful import.
type connectDoneMsg struct{}

// homeConnectMsg is sent when the user launches the browser connector from a
// connect card on the home sidebar. MainModel opens the connect overlay over the
// home view and starts scanning the chosen platform.
type homeConnectMsg struct {
	platform browser.Platform
}

type ViewType int

const (
	ViewHome ViewType = iota
	ViewDownloads
	ViewProfile
	ViewPlaylist
	ViewSettings
)

type MainModel struct {
	view   ViewType
	width  int
	height int
	ready  bool

	home          *HomeModel
	profile       *ProfileModel
	playlist      *PlaylistModel
	downloads     *DownloadsModel
	settings      *SettingsModel
	connect       *ConnectModel
	connectActive bool // when true the connect flow overlays the home view

	activeNav        string
	playerBarFocused bool
	showLangModal    bool
	lang             state.Language
	lastNetCheck     time.Time
}

func NewMainModel() *MainModel {
	m := &MainModel{
		view:          ViewHome,
		activeNav:     "home",
		width:         160,
		height:        50,
		home:          NewHomeModel(),
		profile:       NewProfileModel(),
		playlist:      NewPlaylistModel(),
		downloads:     NewDownloadsModel(),
		settings:      NewSettingsModel(),
		connect:       NewConnectModel(),
		ready:         true,
		showLangModal: state.Current.IsFirstLaunch,
		lang:          state.LangTurkish,
	}
	if len(state.AllLanguages()) > 0 && state.Current.Language != "" {
		m.lang = state.Current.Language
	}
	return m
}

func (m *MainModel) Init() tea.Cmd {
	return tea.Batch(
		tea.HideCursor,
		tea.SetWindowTitle("MusicLe"),
		m.home.Init(),
		m.settings.Init(),
		m.downloads.Init(),
		m.pollTicker(),
		m.spectrumTicker(),
	)
}

func (m *MainModel) pollTicker() tea.Cmd {
	return tea.Every(500*time.Millisecond, func(t time.Time) tea.Msg {
		return PollTickMsg(t)
	})
}

type PollTickMsg time.Time

// SpectrumTickMsg fires on the configured spectrum auto-cycle interval.
type SpectrumTickMsg time.Time

// spectrumTicker schedules the next spectrum color step. The delay is read from
// the live setting on every tick, so changing the speed (or turning the cycle
// off) takes effect without starting a second ticker; while the cycle is off
// the ticker idles at 1s and applies nothing.
func (m *MainModel) spectrumTicker() tea.Cmd {
	ms := state.Current.SpectrumAutoCycleMs
	if ms <= 0 {
		ms = 1000
	}
	return tea.Tick(time.Duration(ms)*time.Millisecond, func(t time.Time) tea.Msg {
		return SpectrumTickMsg(t)
	})
}

// rotateSpectrum advances the spectrum color cycle by one step and reschedules
// itself. The selected palette stays selected: only its own band colors shift,
// and only the spectrum bars show the change. The delay is re-read on every
// tick, so changing the speed (or turning the cycle off) never starts a second
// ticker; while the cycle is off the ticker idles at 1s and applies nothing.
func (m *MainModel) rotateSpectrum() tea.Cmd {
	if state.Current.SpectrumAutoCycleMs > 0 {
		ui.RotateSpectrum()
	} else if ui.SpectrumRotated() {
		// Cycle turned off: reload the selected palette so the spectrum goes
		// back to the exact colors it had before the rotation started.
		ui.SetSpectrumPalette(state.Current.SpectrumPalette)
	}
	return m.spectrumTicker()
}

type PlayerStatusResult struct {
	Result *bridge.Result
	Error  error
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		if m.showLangModal {
			cycleLang := func(dir int) {
				langs := state.AllLanguages()
				idx := 0
				for i, l := range langs {
					if l == m.lang {
						idx = i
						break
					}
				}
				idx = (idx + dir + len(langs)) % len(langs)
				m.lang = langs[idx]
			}
			switch msg.String() {
			case "up", "k":
				cycleLang(-1)
			case "down", "j":
				cycleLang(1)
			case "enter":
				return m, initializeDefaults(m.lang)
			}
			return m, nil
		}

		// While the connect flow overlays the home view, route all key input to
		// the ConnectModel. Esc cancels and closes the overlay.
		if m.connectActive {
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
			if msg.String() == "esc" {
				m.connectActive = false
				m.connect = NewConnectModel()
				return m, nil
			}
			if m.connect != nil {
				newC, cmd := m.connect.Update(msg)
				m.connect = newC.(*ConnectModel)
				if cmd != nil {
					return m, cmd
				}
			}
			return m, nil
		}

		switch {
		case msg.Type == tea.KeyCtrlC:
			if m.view == ViewDownloads && m.downloads != nil {
				m.downloads.HandleCtrlC()
				return m, nil
			}
			return m, tea.Quit
		case m.view == ViewHome && m.home != nil && m.home.inputCaptured():
			// An overlay on the home view (delete/edit/rename prompt, expanded
			// playlist list) owns every key: nothing global may steal the
			// arrows/Enter/Esc it is waiting for.
			newHome, cmd := m.home.Update(msg)
			m.home = newHome.(*HomeModel)
			return m, cmd
		case msg.Type == tea.KeyF1:
			if m.playerBarFocused {
				m.playerBarFocused = false
				switch m.view {
				case ViewHome:
					if m.home != nil {
						m.home.applyRegion(1)
					}
				case ViewProfile:
					if m.profile != nil {
						m.profile.focus = 0
						m.profile.setFocus(0)
					}
				case ViewPlaylist:
					if m.playlist != nil {
						m.playlist.focus = 0
						m.playlist.setFocus(0)
					}
				case ViewSettings:
					if m.settings != nil {
						m.settings.focus = 0
					}
				}
				return m, nil
			}
			var cmd tea.Cmd
			wrapped := false
			switch m.view {
			case ViewHome:
				if m.home != nil {
					wrapped, cmd = m.home.CycleSection()
				}
			case ViewProfile:
				if m.profile != nil {
					wrapped = m.profile.cycleFocus()
				}
			case ViewPlaylist:
				if m.playlist != nil {
					wrapped = m.playlist.cycleFocus()
				}
			case ViewSettings:
				if m.settings != nil {
					wrapped = m.settings.cycleFocus()
				}
			case ViewDownloads:
				if m.downloads != nil {
					wrapped = m.downloads.cycleFocus()
				}
			}
			if wrapped {
				m.playerBarFocused = true
				switch m.view {
				case ViewHome:
					if m.home != nil {
						m.home.sectionFocus = -1
					}
				case ViewProfile:
					if m.profile != nil {
						m.profile.focus = -1
					}
				case ViewPlaylist:
					if m.playlist != nil {
						m.playlist.focus = -1
					}
				case ViewSettings:
					if m.settings != nil {
						m.settings.focus = -1
					}
				}
			}
			if cmd != nil {
				return m, cmd
			}
			return m, nil
		case msg.Type == tea.KeyF2:
			if m.playerBarFocused {
				m.playerBarFocused = false
			}
			m.view = (m.view + 1) % 5
			switch m.view {
			case ViewHome:
				m.activeNav = "home"
				if m.home != nil {
					m.home.refreshAllContent()
				}
			case ViewDownloads:
				m.activeNav = "downloads"
				m.downloads.refreshPlaylistOptions()
			case ViewProfile:
				m.activeNav = "profile"
			case ViewPlaylist:
				m.activeNav = "playlist"
			case ViewSettings:
				m.activeNav = "settings"
			}
			return m, nil
		case msg.Type == tea.KeyF3:
			// Settings tab cycling — only meaningful in the General view.
			if m.view == ViewSettings && m.settings != nil {
				m.settings.cycleTab()
				return m, nil
			}
		case msg.Type == tea.KeyEscape:
			if m.connectActive {
				m.connectActive = false
				return m, nil
			}
			if m.playerBarFocused {
				m.playerBarFocused = false
			}
			if m.view != ViewHome {
				m.view = ViewHome
				m.activeNav = "home"
				if m.home != nil {
					m.home.refreshAllContent()
				}
				return m, nil
			}
		case m.playerBarFocused:
			switch msg.String() {
			case "up":
				v := state.Current.Player.Volume + 0.05
				if v > 1 {
					v = 1
				}
				state.Current.Player.Volume = v
				go bridge.PlayerCall(bridge.Action{Action: "volume", Value: v})
				return m, nil
			case "down":
				v := state.Current.Player.Volume - 0.05
				if v < 0 {
					v = 0
				}
				state.Current.Player.Volume = v
				go bridge.PlayerCall(bridge.Action{Action: "volume", Value: v})
				return m, nil
			case "left":
				go bridge.PlayerCall(bridge.Action{Action: "seek", Value: -5})
				return m, nil
			case "right":
				go bridge.PlayerCall(bridge.Action{Action: "seek", Value: 5})
				return m, nil
			case " ":
				ps := &state.Current.Player
				if ps.CurrentSong == nil {
					if state.Current.CurrentPlaylist != nil && len(state.Current.CurrentPlaylist.Songs) > 0 {
						go bridge.PlayerCall(bridge.Action{Action: "play", File: state.Current.CurrentPlaylist.Songs[0].FilePath})
					}
				} else if ps.IsPlaying {
					go bridge.PlayerCall(bridge.Action{Action: "pause"})
				} else {
					go bridge.PlayerCall(bridge.Action{Action: "resume"})
				}
				return m, nil
			}
		}

	case PollTickMsg:
		cmd := m.handlePlayerPoll()
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if m.home != nil {
			cmd := m.home.checkAutoAdvance()
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		cmds = append(cmds, m.pollTicker())
		m.maybeCheckNetwork()
		if active, pct, status := bridge.CurrentDownload.Get(); m.downloads != nil {
			m.downloads.downloadPercent = pct
			m.downloads.downloadStatus = status
			m.downloads.TrackProgress(active, pct, status)
		}

	case SpectrumTickMsg:
		cmds = append(cmds, m.rotateSpectrum())

	case StartDownloadMsg:
		cmds = append(cmds, m.handleDownload(msg))

	case DownloadResultMsg:
		if m.downloads != nil {
			m.downloads.handleDownloadResult(msg)
		}
		if m.home != nil {
			m.home.refreshAllContent()
		}

	case LocalFileImportMsg:
		cmds = append(cmds, m.handleLocalImport(msg))

	case ImportResultMsg:
		if m.home != nil {
			cmds = append(cmds, m.home.OnImportResult(msg))
		}

	case ThemeChangedMsg:
		if m.downloads != nil {
			m.downloads.RefreshTheme()
		}

	case setupDoneMsg:
		m.showLangModal = false

	case errorMsg:
		m.showLangModal = false

	case ConnectResultMsg:
		if m.connect != nil {
			m.connect.finishScan(msg.Playlists, msg.Err)
		}
		return m, nil

	case connectDoneMsg:
		m.connectActive = false
		m.activeNav = "home"
		if m.home != nil {
			m.home.refreshAllContent()
		}
		return m, nil

	case homeConnectMsg:
		if m.connect == nil {
			m.connect = NewConnectModel()
		}
		m.connect.focus = 0
		if msg.platform == browser.PlatformYouTube {
			m.connect.focus = 1
		}
		m.connect.scanning = true
		m.connect.scanStart = time.Now()
		m.connectActive = true
		return m, scanConnectCmd(msg.platform)
	}

	switch m.view {
	case ViewHome:
		if m.home != nil {
			newHome, cmd := m.home.Update(msg)
			m.home = newHome.(*HomeModel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ViewProfile:
		if m.profile != nil {
			newP, cmd := m.profile.Update(msg)
			m.profile = newP.(*ProfileModel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ViewPlaylist:
		if m.playlist != nil {
			newPl, cmd := m.playlist.Update(msg)
			m.playlist = newPl.(*PlaylistModel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ViewSettings:
		if m.settings != nil {
			newS, cmd := m.settings.Update(msg)
			m.settings = newS.(*SettingsModel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	case ViewDownloads:
		if m.downloads != nil {
			newD, cmd := m.downloads.Update(msg)
			m.downloads = newD.(*DownloadsModel)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *MainModel) handlePlayerPoll() tea.Cmd {
	return func() tea.Msg {
		result, err := bridge.PlayerCall(bridge.Action{Action: "status"})
		return PlayerStatusResult{Result: result, Error: err}
	}
}

func (m *MainModel) handleDownload(msg StartDownloadMsg) tea.Cmd {
	return func() tea.Msg {
		result, err := bridge.RunScriptDownload(bridge.Action{
			Action: msg.Action,
			URL:    msg.URL,
			Output: msg.Output,
		})
		return DownloadResultMsg{Result: result, Error: err, Action: msg.Action}
	}
}

func (m *MainModel) handleLocalImport(msg LocalFileImportMsg) tea.Cmd {
	return func() tea.Msg {
		result, err := bridge.RunScript(bridge.Action{
			Action: "add_local",
			File:   msg.FilePath,
			Output: msg.Output,
		})
		return ImportResultMsg{Result: result, Error: err}
	}
}

func (m *MainModel) maybeCheckNetwork() {
	if time.Since(m.lastNetCheck) < 30*time.Second {
		return
	}
	m.lastNetCheck = time.Now()
	conn, err := net.DialTimeout("tcp", "google.com:80", 2*time.Second)
	if err == nil {
		conn.Close()
		state.Current.NetworkOnline = true
	} else {
		state.Current.NetworkOnline = false
	}
}

func (m *MainModel) View() string {
	if !m.ready {
		return "Loading..."
	}

	header := components.RenderHeader(m.width, m.activeNav)
	playerBar := components.RenderPlayerBar(m.width, m.playerBarFocused)

	headerH := lipgloss.Height(header)
	barH := lipgloss.Height(playerBar)
	bodyH := m.height - headerH - barH
	if bodyH < 5 {
		bodyH = 5
	}

	body := ""
	switch m.view {
	case ViewHome:
		if m.home != nil {
			oldH := m.home.height
			m.home.height = bodyH
			body = m.home.View()
			m.home.height = oldH
		}
	case ViewProfile:
		if m.profile != nil {
			body = m.profile.View()
		}
	case ViewPlaylist:
		if m.playlist != nil {
			body = m.playlist.View()
		}
	case ViewSettings:
		if m.settings != nil {
			m.settings.width = m.width
			m.settings.height = bodyH
			body = m.settings.View()
		}
	case ViewDownloads:
		if m.downloads != nil {
			m.downloads.width = m.width
			m.downloads.height = bodyH
			body = m.downloads.View()
		}
	}

	if m.connectActive && m.connect != nil {
		m.connect.width = m.width
		m.connect.height = bodyH
		overlay := lipgloss.NewStyle().
			Width(m.width).
			Height(bodyH).
			Background(ui.ColorBackground).
			Render(m.connect.View())
		body = placeOverlay(body, overlay, m.width)
	}
	if m.view != ViewHome {
		body = lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(body)
	}
	bodyHActual := lipgloss.Height(body)
	if bodyHActual < bodyH {
		if m.view != ViewHome {
			top := (bodyH - bodyHActual) / 2
			bottom := bodyH - bodyHActual - top
			body = strings.Repeat("\n", top) + body + strings.Repeat("\n", bottom)
		} else {
			body += strings.Repeat("\n", bodyH-bodyHActual)
		}
	} else if bodyHActual > bodyH {
		// Trim excess lines to prevent overflow
		lines := strings.Split(body, "\n")
		if len(lines) > bodyH {
			body = strings.Join(lines[:bodyH], "\n")
		}
	}

	full := lipgloss.JoinVertical(lipgloss.Left, header, body, playerBar)

	if m.showLangModal {
		modal := renderLangModal(m.lang)
		full = placeOverlay(full, modal, m.width)
	}

	return full
}
