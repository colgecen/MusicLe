package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MusicLe/state"
	"MusicLe/ui"
)

type PlaylistModel struct {
	width  int
	height int

	leftColWidth int

	playlistFocusIdx int
	playlistOffset   int
	playlistOptions  []string

	plNameInput textinput.Model
	plBioInput  textinput.Model

	addMode bool

	playlistStatus string

	focus       int
	lastProfile string
	selectAll   bool
}

func NewPlaylistModel() *PlaylistModel {
	return &PlaylistModel{
		leftColWidth: 30,
		plNameInput: func() textinput.Model {
			ti := textinput.New()
			ti.Prompt = Tr("pl.name_prompt")
			ti.Placeholder = Tr("pl.name_ph")
			ti.Width = 60
			return ti
		}(),
		plBioInput: func() textinput.Model {
			ti := textinput.New()
			ti.Prompt = Tr("pl.desc_prompt")
			ti.Placeholder = Tr("pl.desc_ph")
			ti.Width = 60
			return ti
		}(),
	}
}

func (m *PlaylistModel) Init() tea.Cmd { return nil }

func (m *PlaylistModel) refreshOptions() {
	cp := state.Current.CurrentProfile
	if cp == nil {
		return
	}
	opts := make([]string, len(cp.Playlists))
	for i, pl := range cp.Playlists {
		opts[i] = fmt.Sprintf("%d. %s", i+1, pl.Name)
	}
	m.playlistOptions = opts
	if m.playlistFocusIdx >= len(opts) {
		m.playlistFocusIdx = 0
		m.playlistOffset = 0
	}
	pl := m.selectedPlaylist()
	if pl != nil {
		id := cp.FolderName + "/" + pl.FolderName
		if id != m.lastProfile {
			m.lastProfile = id
			m.plNameInput.SetValue(pl.Name)
			m.plNameInput.SetCursor(len(pl.Name))
			m.plBioInput.SetValue(pl.Bio)
			m.plBioInput.SetCursor(len(pl.Bio))
		}
	}
}

func (m *PlaylistModel) selectedPlaylist() *state.Playlist {
	cp := state.Current.CurrentProfile
	if cp == nil {
		return nil
	}
	if m.playlistFocusIdx >= 0 && m.playlistFocusIdx < len(cp.Playlists) {
		return &cp.Playlists[m.playlistFocusIdx]
	}
	return nil
}

func (m *PlaylistModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.refreshOptions()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// Focus map: 0=list, 1=name, 2=bio, 3=save/create, 4=delete, 5=add.
		switch msg.String() {
		case "up", "k":
			if m.focus == 0 {
				if m.playlistFocusIdx > 0 {
					m.playlistFocusIdx--
				}
			} else if m.focus >= 1 && m.focus <= 2 {
				m.selectAll = false
				inputs := []*textinput.Model{&m.plNameInput, &m.plBioInput}
				var cmd tea.Cmd
				*inputs[m.focus-1], cmd = inputs[m.focus-1].Update(msg)
				return m, cmd
			}
		case "down", "j":
			if m.focus == 0 {
				if m.playlistFocusIdx < len(m.playlistOptions)-1 {
					m.playlistFocusIdx++
				}
			} else if m.focus >= 1 && m.focus <= 2 {
				m.selectAll = false
				inputs := []*textinput.Model{&m.plNameInput, &m.plBioInput}
				var cmd tea.Cmd
				*inputs[m.focus-1], cmd = inputs[m.focus-1].Update(msg)
				return m, cmd
			}
		case "tab":
			m.selectAll = false
			m.setFocus((m.focus + 1) % 6)
		case "shift+tab":
			m.selectAll = false
			m.setFocus((m.focus - 1 + 6) % 6)
		case "enter":
			if m.focus == 0 {
				m.selectPlaylist()
			} else if m.focus == 3 {
				if m.addMode {
					return m.addNewPlaylist()
				}
				return m.savePlaylist()
			} else if m.focus == 4 {
				return m.deleteCurrentPlaylist()
			} else if m.focus == 5 {
				m.enterAddMode()
			}
		case "delete":
			if m.focus == 4 {
				return m.deleteCurrentPlaylist()
			}
		case "esc":
			if m.addMode {
				m.cancelAddMode()
			} else {
				m.setFocus(0)
			}
		case "ctrl+v":
			if m.focus >= 1 && m.focus <= 2 {
				inputs := []*textinput.Model{&m.plNameInput, &m.plBioInput}
				*inputs[m.focus-1], _ = inputs[m.focus-1].Update(textinput.Paste())
				return m, nil
			}
		case "ctrl+a":
			if m.focus >= 1 && m.focus <= 2 {
				inputs := []*textinput.Model{&m.plNameInput, &m.plBioInput}
				if inputs[m.focus-1].Value() != "" {
					m.selectAll = true
				}
			}
			return m, nil
		default:
			if m.focus >= 1 && m.focus <= 2 {
				if m.selectAll {
					inp := []*textinput.Model{&m.plNameInput, &m.plBioInput}[m.focus-1]
					s := msg.String()
					if len(s) == 1 || s == "backspace" || s == "delete" {
						inp.SetValue("")
						inp.SetCursor(0)
						m.selectAll = false
					} else {
						m.selectAll = false
					}
				}
				inputs := []*textinput.Model{&m.plNameInput, &m.plBioInput}
				var cmd tea.Cmd
				*inputs[m.focus-1], cmd = inputs[m.focus-1].Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m *PlaylistModel) setFocus(idx int) {
	if idx < 0 || idx >= 6 {
		return
	}
	m.focus = idx
	inputs := []*textinput.Model{&m.plNameInput, &m.plBioInput}
	for i, inp := range inputs {
		if i+1 == idx {
			inp.Focus()
		} else {
			inp.Blur()
		}
	}
}

func (m *PlaylistModel) cycleFocus() bool {
	if m.focus == 0 {
		m.setFocus(3)
	} else {
		if m.addMode {
			m.cancelAddMode()
		}
		m.setFocus(0)
	}
	return false
}

func (m *PlaylistModel) selectPlaylist() {
	cp := state.Current.CurrentProfile
	if cp == nil {
		return
	}
	if m.playlistFocusIdx >= 0 && m.playlistFocusIdx < len(cp.Playlists) {
		state.Current.CurrentPlaylist = &cp.Playlists[m.playlistFocusIdx]
		m.cancelAddMode()
		m.refreshOptions()
	}
}

func (m *PlaylistModel) enterAddMode() {
	m.addMode = true
	m.plNameInput.SetValue("")
	m.plBioInput.SetValue("")
	m.playlistStatus = ""
	m.setFocus(1)
}

func (m *PlaylistModel) cancelAddMode() {
	m.addMode = false
	pl := m.selectedPlaylist()
	if pl != nil {
		m.plNameInput.SetValue(pl.Name)
		m.plNameInput.SetCursor(len(pl.Name))
		m.plBioInput.SetValue(pl.Bio)
		m.plBioInput.SetCursor(len(pl.Bio))
	}
	m.playlistStatus = ""
	m.setFocus(0)
}

func (m *PlaylistModel) addNewPlaylist() (tea.Model, tea.Cmd) {
	cp := state.Current.CurrentProfile
	if cp == nil {
		m.playlistStatus = ui.ErrorStyle.Render("  x " + Tr("pl.no_profile"))
		return m, nil
	}
	name := strings.TrimSpace(m.plNameInput.Value())
	if name == "" {
		m.playlistStatus = ui.ErrorStyle.Render("  x " + Tr("pl.name_required"))
		return m, nil
	}
	for _, pl := range cp.Playlists {
		if pl.Name == name {
			m.playlistStatus = ui.ErrorStyle.Render("  x " + Tr("pl.name_exists"))
			return m, nil
		}
	}
	folder := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	bio := strings.TrimSpace(m.plBioInput.Value())
	// Remember which playlist was active: creating a new one must not steal
	// the selection shown on the home screen.
	prevFolder := ""
	if state.Current.CurrentPlaylist != nil {
		prevFolder = state.Current.CurrentPlaylist.FolderName
	}
	if err := state.Current.CreatePlaylistStructure(cp.FolderName, folder, name, bio, ""); err != nil {
		m.playlistStatus = ui.ErrorStyle.Render("  x Error: " + err.Error())
		return m, nil
	}
	_ = state.Current.ScanProfiles()
	for i, p := range state.Current.Profiles {
		if p.FolderName != cp.FolderName {
			continue
		}
		state.Current.CurrentProfile = &state.Current.Profiles[i]
		state.Current.CurrentPlaylist = nil
		for j, pl := range p.Playlists {
			if pl.FolderName == prevFolder {
				state.Current.CurrentPlaylist = &p.Playlists[j]
				break
			}
		}
		newIdx := -1
		for j, pl := range p.Playlists {
			if pl.FolderName == folder {
				newIdx = j
				break
			}
		}
		if newIdx < 0 {
			break
		}
		// First playlist of the profile: there was no selection to keep.
		if state.Current.CurrentPlaylist == nil {
			state.Current.CurrentPlaylist = &p.Playlists[newIdx]
		}
		// The cursor in this view lands on the playlist just created; the
		// active playlist on the home screen stays where it was.
		m.playlistFocusIdx = newIdx
		break
	}
	m.addMode = false
	m.refreshOptions()
	m.playlistStatus = ui.AccentStyle.Render("  v " + Tr("pl.created"))
	m.setFocus(0) // return to first button after creation
	return m, nil
}

func (m *PlaylistModel) savePlaylist() (tea.Model, tea.Cmd) {
	cp := state.Current.CurrentProfile
	if cp == nil {
		return m, nil
	}
	pl := m.selectedPlaylist()
	if pl == nil {
		return m, nil
	}
	name := strings.TrimSpace(m.plNameInput.Value())
	if name == "" {
		name = pl.FolderName
	}
	bio := strings.TrimSpace(m.plBioInput.Value())
	if err := state.Current.SavePlaylistMeta(cp.FolderName, pl.FolderName, name, bio); err != nil {
		m.playlistStatus = ui.ErrorStyle.Render("  x Error: " + err.Error())
		return m, nil
	}
	_ = state.Current.ScanProfiles()
	for i, p := range state.Current.Profiles {
		if p.FolderName == cp.FolderName {
			state.Current.CurrentProfile = &state.Current.Profiles[i]
			if m.playlistFocusIdx < len(p.Playlists) {
				state.Current.CurrentPlaylist = &p.Playlists[m.playlistFocusIdx]
			}
			break
		}
	}
	m.refreshOptions()
	m.playlistStatus = ui.AccentStyle.Render("  v " + Tr("pl.saved"))
	return m, nil
}

func (m *PlaylistModel) deleteCurrentPlaylist() (tea.Model, tea.Cmd) {
	cp := state.Current.CurrentProfile
	pl := m.selectedPlaylist()
	if cp == nil || pl == nil {
		return m, nil
	}
	_ = state.Current.DeletePlaylist(cp.FolderName, pl.FolderName)
	_ = state.Current.ScanProfiles()
	for i, p := range state.Current.Profiles {
		if p.FolderName == cp.FolderName {
			state.Current.CurrentProfile = &state.Current.Profiles[i]
			if len(p.Playlists) > 0 {
				idx := m.playlistFocusIdx
				if idx >= len(p.Playlists) {
					idx = len(p.Playlists) - 1
				}
				state.Current.CurrentPlaylist = &p.Playlists[idx]
				m.playlistFocusIdx = idx
			} else {
				state.Current.CurrentPlaylist = nil
				m.playlistFocusIdx = 0
			}
			break
		}
	}
	m.refreshOptions()
	if m.playlistFocusIdx >= len(m.playlistOptions) {
		m.playlistFocusIdx = 0
	}
	m.playlistStatus = ui.DimStyle.Render("  " + Tr("pl.deleted"))
	return m, nil
}

func (m *PlaylistModel) View() string {
	if m.width <= 0 {
		m.width = 120
	}
	if m.height <= 0 {
		m.height = 40
	}

	leftW := m.leftColWidth
	if leftW < 20 {
		leftW = 20
	}
	rightW := m.width - leftW - 4
	if rightW < 40 {
		rightW = 40
	}

	rightPanel := m.renderRightPanel(rightW)
	rightH := lipgloss.Height(rightPanel)
	leftPanel := m.renderLeftPanel(leftW, rightH)

	leftH := lipgloss.Height(leftPanel)
	if leftH < rightH {
		leftPanel += strings.Repeat("\n", rightH-leftH)
	} else if rightH < leftH {
		rightPanel += strings.Repeat("\n", leftH-rightH)
	}

	joined := lipgloss.JoinHorizontal(lipgloss.Top,
		leftPanel,
		"  ",
		rightPanel,
	)

	return joined
}

func (m *PlaylistModel) renderLeftPanel(w, maxH int) string {
	innerH := maxH - 4
	if innerH < 3 {
		innerH = 3
	}
	maxVisible := innerH - 1
	if maxVisible < 1 {
		maxVisible = 1
	}

	total := len(m.playlistOptions)

	if m.playlistFocusIdx < m.playlistOffset {
		m.playlistOffset = m.playlistFocusIdx
	}
	if m.playlistFocusIdx >= m.playlistOffset+maxVisible {
		m.playlistOffset = m.playlistFocusIdx - maxVisible + 1
	}
	if m.playlistOffset > total-maxVisible && m.playlistOffset > 0 {
		m.playlistOffset = total - maxVisible
	}
	if m.playlistOffset < 0 {
		m.playlistOffset = 0
	}

	var lines []string
	if total == 0 {
		lines = append(lines, ui.DimStyle.Render("  "+Tr("pl.no_playlists")))
	} else {
		end := m.playlistOffset + maxVisible
		if end > total {
			end = total
		}
		for i := m.playlistOffset; i < end; i++ {
			item := "  " + m.playlistOptions[i]
			if m.focus == 0 && i == m.playlistFocusIdx {
				item = ui.AccentStyle.Render("> " + m.playlistOptions[i])
			}
			lines = append(lines, item)
		}
	}

	content := strings.Join(lines, "\n")
	contentH := lipgloss.Height(content)
	if contentH < innerH {
		content += strings.Repeat("\n", innerH-contentH)
	}

	title := ui.SectionTitleStyle.Render(Tr("pl.playlists_title"))
	box := ui.AccentBorderStyle.
		Width(w).
		Height(maxH - 2).
		Render(title + "\n" + content)

	return box
}

func (m *PlaylistModel) renderRightPanel(w int) string {
	plV := "-"
	pl := m.selectedPlaylist()
	if pl != nil {
		plV = pl.Name
	}

	titlePrefix := Tr("pl.playlists_title")
	if m.addMode {
		titlePrefix = Tr("pl.new_prefix")
		plV = Tr("pl.creating_new")
	}

	plNameVal := m.plNameInput.Value()
	if plNameVal == "" && !m.addMode {
		plNameVal = m.plNameInput.Placeholder
	}
	var plNameV string
	if m.focus == 1 {
		plNameV = m.plNameInput.View()
	} else {
		plNameV = Tr("pl.name_prompt") + ui.WhiteStyle.Render(plNameVal)
	}

	plBioVal := m.plBioInput.Value()
	if plBioVal == "" && !m.addMode {
		plBioVal = m.plBioInput.Placeholder
	}
	var plBioV string
	if m.focus == 2 {
		plBioV = m.plBioInput.View()
	} else {
		plBioV = Tr("pl.desc_prompt") + ui.WhiteStyle.Render(plBioVal)
	}

	// Button labels keep the width of their longest translation across all
	// 11 languages, so the row never shifts when the language changes.
	saveW := maxTrWidth("pl.save")
	if w := maxTrWidth("pl.create"); w > saveW {
		saveW = w
	}
	delW := maxTrWidth("pl.delete_btn")
	addW := maxTrWidth("pl.add_btn")
	saveLabel := ui.FitPad(Tr("pl.save"), saveW)
	if m.addMode {
		saveLabel = ui.FitPad(Tr("pl.create"), saveW)
	}
	delLabel := ui.FitPad(Tr("pl.delete_btn"), delW)
	addLabel := ui.FitPad(Tr("pl.add_btn"), addW)
	saveBtn := ui.AccentButtonStyle.Render(saveLabel)
	deleteBtn := ui.ErrorButtonStyle.Render(delLabel)
	addBtn := ui.ButtonStyle.Render(addLabel)

	if m.focus == 3 {
		saveBtn = ui.FocusedButtonStyle.Render(saveLabel)
	}
	if m.focus == 4 {
		deleteBtn = ui.FocusedButtonStyle.Render(delLabel)
	}
	if m.focus == 5 {
		addBtn = ui.FocusedButtonStyle.Render(addLabel)
	}

	boxContent := lipgloss.JoinVertical(lipgloss.Left,
		"",
		ui.SectionTitleStyle.Render(" "+titlePrefix+": ")+plV,
		"",
		ui.SectionTitleStyle.Render(Tr("pl.sec_name")),
		"",
		plNameV,
		"",
		ui.SectionTitleStyle.Render(Tr("pl.sec_desc")),
		"",
		plBioV,
		"",
		m.playlistStatus,
		"",
		lipgloss.JoinHorizontal(lipgloss.Left, saveBtn, "  ", deleteBtn, "  ", addBtn),
	)

	title := ui.SectionTitleStyle.Render(Tr("pl.settings_title"))
	box := ui.AccentBorderStyle.
		Width(w).
		Render(title + "\n" + boxContent)

	return box
}
