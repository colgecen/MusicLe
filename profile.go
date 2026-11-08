package main

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MusicLe/state"
	"MusicLe/ui"
)

type ProfileModel struct {
	width  int
	height int

	profileDropIdx int
	profileOptions []string
	nameInput      textinput.Model
	bioInput       textinput.Model
	langIdx        int
	profileStatus  string

	focus       int
	lastProfile string // tracks last loaded profile folder to avoid re-populating inputs
	selectAll   bool
}

func NewProfileModel() *ProfileModel {
	return &ProfileModel{
		langIdx: func() int {
			for i, l := range state.AllLanguages() {
				if l == state.Current.Language {
					return i
				}
			}
			return 0
		}(),
		nameInput: func() textinput.Model {
			ti := textinput.New()
			ti.Prompt = Tr("profile.name_prompt")
			ti.Placeholder = "MusicLe User"
			ti.Width = 60
			return ti
		}(),
		bioInput: func() textinput.Model {
			ti := textinput.New()
			ti.Prompt = Tr("profile.bio_prompt")
			ti.Placeholder = Tr("profile.bio_ph")
			ti.Width = 60
			return ti
		}(),
	}
}

func (m *ProfileModel) Init() tea.Cmd { return nil }

func (m *ProfileModel) refreshOptions() {
	state.Current.ScanProfiles()
	opts := make([]string, len(state.Current.Profiles))
	for i, p := range state.Current.Profiles {
		opts[i] = p.DisplayName
	}
	m.profileOptions = opts
	if m.profileDropIdx >= len(opts) {
		m.profileDropIdx = 0
	}
	if len(opts) > 0 {
		state.Current.CurrentProfile = &state.Current.Profiles[m.profileDropIdx]
		cp := state.Current.CurrentProfile
		if cp != nil && cp.FolderName != m.lastProfile {
			m.lastProfile = cp.FolderName
			m.nameInput.SetValue(cp.DisplayName)
			m.nameInput.SetCursor(len(cp.DisplayName))
			m.bioInput.SetValue(cp.Bio)
			m.bioInput.SetCursor(len(cp.Bio))
		}
	}
}

func (m *ProfileModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.refreshOptions()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.focus == 0 {
				if m.profileDropIdx > 0 {
					m.profileDropIdx--
					m.refreshOptions()
				}
			} else if m.focus >= 1 && m.focus <= 3 {
				m.selectAll = false
				inputs := []*textinput.Model{&m.nameInput, &m.bioInput}
				var cmd tea.Cmd
				*inputs[m.focus-1], cmd = inputs[m.focus-1].Update(msg)
				return m, cmd
			}
		case "down", "j":
			if m.focus == 0 {
				if m.profileDropIdx < len(m.profileOptions)-1 {
					m.profileDropIdx++
					m.refreshOptions()
				}
			} else if m.focus >= 1 && m.focus <= 3 {
				m.selectAll = false
				inputs := []*textinput.Model{&m.nameInput, &m.bioInput}
				var cmd tea.Cmd
				*inputs[m.focus-1], cmd = inputs[m.focus-1].Update(msg)
				return m, cmd
			}
		case "left", "h", "right", "l":
			// Cycle profile language (only outside text inputs).
			if m.focus == 0 || m.focus == 3 {
				langs := state.AllLanguages()
				dir := 1
				if msg.String() == "left" || msg.String() == "h" {
					dir = -1
				}
				m.langIdx = (m.langIdx + dir + len(langs)) % len(langs)
				return m, nil
			}
			if m.focus >= 1 && m.focus <= 3 {
				inputs := []*textinput.Model{&m.nameInput, &m.bioInput}
				var cmd tea.Cmd
				*inputs[m.focus-1], cmd = inputs[m.focus-1].Update(msg)
				return m, cmd
			}
		case "tab":
			m.selectAll = false
			m.setFocus((m.focus + 1) % 4)
		case "shift+tab":
			m.selectAll = false
			m.setFocus((m.focus - 1 + 4) % 4)
		case "enter":
			if m.focus == 3 {
				cp := state.Current.CurrentProfile
				if cp == nil {
					return m, nil
				}
				name := strings.TrimSpace(m.nameInput.Value())
				if name == "" {
					name = cp.FolderName
				}
				bio := strings.TrimSpace(m.bioInput.Value())
				if err := state.Current.SaveProfileMeta(cp.FolderName, name, bio); err != nil {
					m.profileStatus = ui.ErrorStyle.Render("  x Error: " + err.Error())
					return m, nil
				}
				langs := state.AllLanguages()
				lang := langs[m.langIdx%len(langs)]
				state.Current.Language = lang
				_ = state.Current.SaveConfig()
				_ = state.Current.ScanProfiles()
				for i, p := range state.Current.Profiles {
					if p.FolderName == cp.FolderName {
						state.Current.CurrentProfile = &state.Current.Profiles[i]
						break
					}
				}
				m.refreshOptions()
				m.profileStatus = ui.AccentStyle.Render("  v " + Tr("pl.saved"))
			}
		case "esc":
			m.focus = 0
		case "ctrl+v":
			if m.focus >= 1 && m.focus <= 3 {
				inputs := []*textinput.Model{&m.nameInput, &m.bioInput}
				*inputs[m.focus-1], _ = inputs[m.focus-1].Update(textinput.Paste())
				return m, nil
			}
		case "ctrl+a":
			if m.focus >= 1 && m.focus <= 3 {
				inputs := []*textinput.Model{&m.nameInput, &m.bioInput}
				if inputs[m.focus-1].Value() != "" {
					m.selectAll = true
				}
			}
			return m, nil
		default:
			if m.focus >= 1 && m.focus <= 3 {
				if m.selectAll {
					inp := []*textinput.Model{&m.nameInput, &m.bioInput}[m.focus-1]
					s := msg.String()
					if len(s) == 1 || s == "backspace" || s == "delete" {
						inp.SetValue("")
						inp.SetCursor(0)
						m.selectAll = false
					} else {
						m.selectAll = false
					}
				}
				inputs := []*textinput.Model{&m.nameInput, &m.bioInput}
				var cmd tea.Cmd
				*inputs[m.focus-1], cmd = inputs[m.focus-1].Update(msg)
				return m, cmd
			}
		}
	}
	return m, nil
}

func (m *ProfileModel) setFocus(idx int) {
	if idx < 0 || idx >= 4 {
		return
	}
	m.focus = idx
	inputs := []*textinput.Model{&m.nameInput, &m.bioInput}
	for i, inp := range inputs {
		if i+1 == idx {
			inp.Focus()
		} else {
			inp.Blur()
		}
	}
}

func (m *ProfileModel) cycleFocus() bool {
	m.setFocus((m.focus + 1) % 4)
	return m.focus == 0
}

func (m *ProfileModel) View() string {
	if m.width <= 0 {
		m.width = 120
	}
	if m.height <= 0 {
		m.height = 40
	}

	profileV := "-"
	if len(m.profileOptions) > 0 && m.profileDropIdx < len(m.profileOptions) {
		profileV = m.profileOptions[m.profileDropIdx]
	}
	if m.focus == 0 {
		profileV = ui.AccentStyle.Render("> " + profileV)
	} else {
		profileV = "  " + ui.WhiteStyle.Render(profileV)
	}

	nameVal := m.nameInput.Value()
	if nameVal == "" {
		nameVal = m.nameInput.Placeholder
	}
	var nameV string
	if m.focus == 1 {
		nameV = m.nameInput.View()
	} else {
		nameV = Tr("profile.name_prompt") + ui.WhiteStyle.Render(nameVal)
	}

	bioVal := m.bioInput.Value()
	if bioVal == "" {
		bioVal = m.bioInput.Placeholder
	}
	var bioV string
	if m.focus == 2 {
		bioV = m.bioInput.View()
	} else {
		bioV = Tr("profile.bio_prompt") + ui.WhiteStyle.Render(bioVal)
	}

	langs := state.AllLanguages()
	langOpts := state.LanguageEndonym(langs[m.langIdx%len(langs)])

	saveW := maxTrWidth("profile.save")
	saveTxt := ui.FitPad(Tr("profile.save"), saveW)
	boxContent := lipgloss.JoinVertical(lipgloss.Left,
		"",
		ui.SectionTitleStyle.Render(" "+Tr("profile.title")+": ")+profileV,
		"",
		ui.SectionTitleStyle.Render(Tr("profile.sec_name")),
		"",
		nameV,
		"",
		ui.SectionTitleStyle.Render(Tr("profile.sec_bio")),
		"",
		bioV,
		"",
		ui.SectionTitleStyle.Render(Tr("profile.sec_lang"))+ui.WhiteStyle.Render(langOpts),
		"",
		m.profileStatus,
		"",
		func() string {
			btn := ui.AccentButtonStyle.Render(saveTxt)
			if m.focus == 3 {
				btn = ui.FocusedButtonStyle.Render(saveTxt)
			}
			return btn
		}(),
	)

	title := ui.SectionTitleStyle.Render(Tr("profile.settings"))

	// Fill available height so it matches playlist page height
	contentH := m.height - 4
	if contentH < 10 {
		contentH = 10
	}
	boxH := lipgloss.Height(title + "\n" + boxContent)
	if boxH < contentH {
		boxContent += strings.Repeat("\n", contentH-boxH)
	}

	box := ui.AccentBorderStyle.
		Width(75).
		Render(title + "\n" + boxContent)

	return box
}
