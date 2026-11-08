package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MusicLe/internal/audio"
	"MusicLe/state"
	"MusicLe/ui"
)

var themeNames = func() []string {
	names := make([]string, 0, len(ui.ThemeColors))
	for n := range ui.ThemeColors {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}()

// settingsTab describes one of the buttons in the left panel.
type settingsTab struct {
	id string
	en string
	tr string
}

var settingsTabs = []settingsTab{
	{"tab.theme", "Theme", "Tema"},
	{"tab.language", "Language", "Dil"},
	{"tab.sound", "Sound", "Ses"},
	{"tab.extras", "Extras", "Ekstralar"},
	{"tab.policies", "Policies", "Politikalar"},
	{"tab.about", "About", "Hakkinda"},
}

type SettingsModel struct {
	width  int
	height int

	langIdx  int
	themeIdx int

	// Custom-theme hex editor state (last row of the Theme tab).
	editingCustom bool
	customHex     textinput.Model
	customErr     string

	// activeTab is the index into settingsTabs that is currently shown.
	activeTab int
	// rightFocused is true while the right panel (selection list) has focus.
	// When false, keys flow through to the MainModel for F1/F2/F3 handling.
	rightFocused bool
	// focus is kept in sync with MainModel F1 player-bar cycling (-1 = bar).
	focus int

	// Sound tab state
	soundDevices []audio.Device // enumerated output sinks
	soundSel     int            // highlighted device index
	volLimit     int            // 0-100, mirrors state.Current.SoundVolumeLimit

	// Extras tab state
	spectrumIdx int // highlighted spectrum palette index

	// Scroll offset for the long-text tabs (Policies, About).
	scroll int
}

func NewSettingsModel() *SettingsModel {
	m := &SettingsModel{}
	for i, l := range state.AllLanguages() {
		if l == state.Current.Language {
			m.langIdx = i
			break
		}
	}
	for i, n := range themeNames {
		if n == state.Current.Theme {
			m.themeIdx = i
			break
		}
	}
	// A saved custom hex lands on the Custom row.
	if _, ok := ui.ThemeColors[state.Current.Theme]; !ok {
		if _, ok := ui.ParseHexColor(state.Current.Theme); ok {
			m.themeIdx = len(themeNames)
		}
	}
	hexInput := textinput.New()
	hexInput.Placeholder = "RRGGBB"
	hexInput.Prompt = "# "
	hexInput.Width = 20
	hexInput.CharLimit = 18
	m.customHex = hexInput
	m.volLimit = state.Current.SoundVolumeLimit
	if m.volLimit <= 0 || m.volLimit > 100 {
		m.volLimit = 100
	}
	names := ui.SpectrumPaletteNames()
	for i, n := range names {
		if n == state.Current.SpectrumPalette {
			m.spectrumIdx = i
			break
		}
	}
	m.soundDevices = audio.ListOutputDevices()
	// Pre-select the configured device if present.
	if state.Current.SoundOutputDevice != "" {
		for i, d := range m.soundDevices {
			if d.Name == state.Current.SoundOutputDevice {
				m.soundSel = i
				break
			}
		}
	}
	return m
}

func (m *SettingsModel) Init() tea.Cmd { return nil }

// tabLabel returns the localized label for a tab index.
func (m *SettingsModel) tabLabel(i int) string {
	t := settingsTabs[i]
	return Tr(t.id)
}

func (m *SettingsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		// When the right panel is focused, keys navigate the selection list
		// instead of bubbling up to the MainModel global handlers.
		if m.rightFocused {
			if handled, cmd := m.handleScrollKey(msg.String()); handled {
				return m, cmd
			}
			switch msg.String() {
			case "esc", "tab", "shift+tab":
				if m.editingCustom {
					m.editingCustom = false
					m.customHex.Blur()
					m.customErr = ""
					if msg.String() != "esc" {
						m.rightFocused = false
					}
					return m, nil
				}
				m.rightFocused = false
				return m, nil
			}
			// Custom hex editor (last row of the Theme tab) captures keys.
			if m.editingCustom && settingsTabs[m.activeTab].id == "tab.theme" {
				switch msg.String() {
				case "enter":
					if hex, ok := ui.ParseHexColor(m.customHex.Value()); ok {
						m.editingCustom = false
						m.customHex.Blur()
						m.customErr = ""
						state.Current.Theme = hex
						_ = state.Current.SaveConfig()
						ui.ApplyTheme(hex)
						return m, func() tea.Msg { return ThemeChangedMsg{} }
					}
					m.customErr = Tr("theme.invalid")
					return m, nil
				case "up", "k":
					m.editingCustom = false
					m.customHex.Blur()
					m.customErr = ""
					m.moveSelection(-1)
					return m, nil
				case "down", "j":
					m.editingCustom = false
					m.customHex.Blur()
					m.customErr = ""
					m.moveSelection(1)
					return m, nil
				}
				var cmd tea.Cmd
				m.customHex, cmd = m.customHex.Update(msg)
				return m, cmd
			}
			// The Sound tab mixes a device list (up/down) with a volume-limit
			// slider (left/right), so it needs its own key map.
			if settingsTabs[m.activeTab].id == "tab.sound" {
				switch msg.String() {
				case "up", "k":
					m.moveSound(-1)
				case "down", "j":
					m.moveSound(1)
				case "left", "h":
					m.adjustVolLimit(-5)
				case "right", "l":
					m.adjustVolLimit(5)
				case "enter":
					return m, m.applyActiveTab()
				}
				return m, nil
			}
			switch msg.String() {
			case "up", "k":
				m.moveSelection(-1)
				return m, nil
			case "down", "j":
				m.moveSelection(1)
				return m, nil
			case "left", "right":
				// Allow horizontal cycling too — handy for single-row lists.
				m.moveSelection(1)
				return m, nil
			case "enter":
				return m, m.applyActiveTab()
			}
			// Swallow other keys while focused so they don't leak out.
			return m, nil
		}

		switch msg.String() {
		case "enter", "tab", "shift+tab":
			// Enter or Tab enters the right-panel selection list.
			if m.hasSelectionList() {
				m.rightFocused = true
				return m, nil
			}
			return m, m.applyActiveTab()
		}
	}
	// Long-text tabs (Policies, About) scroll with ↑/↓ even when the right
	// panel is not focused.
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if handled, cmd := m.handleScrollKey(keyMsg.String()); handled {
			return m, cmd
		}
	}
	return m, nil
}

// hasSelectionList reports whether the active tab owns a navigable list.
func (m *SettingsModel) hasSelectionList() bool {
	switch settingsTabs[m.activeTab].id {
	case "tab.theme", "tab.language", "tab.sound", "tab.extras":
		return true
	}
	return false
}

// themeRowCount is the preset count plus the trailing Custom row.
func themeRowCount() int { return len(themeNames) + 1 }

// moveSelection shifts the highlighted item of the active tab's list.
func (m *SettingsModel) moveSelection(dir int) {
	m.editingCustom = false
	m.customErr = ""
	switch settingsTabs[m.activeTab].id {
	case "tab.theme":
		n := themeRowCount()
		if n == 0 {
			return
		}
		m.themeIdx = (m.themeIdx + dir + n) % n
	case "tab.language":
		langs := state.AllLanguages()
		m.langIdx = (m.langIdx + dir + len(langs)) % len(langs)
	case "tab.extras":
		// One extra row below the palette list: the auto-cycle toggle.
		n := len(ui.SpectrumPaletteNames()) + 1
		m.spectrumIdx = (m.spectrumIdx + dir + n) % n
	}
}

// moveSound shifts the highlighted device list in the Sound tab.
func (m *SettingsModel) moveSound(dir int) {
	n := len(m.soundDevices)
	if n == 0 {
		return
	}
	m.soundSel = (m.soundSel + dir + n) % n
}

// adjustVolLimit changes the volume cap by step, clamped to 5-100.
func (m *SettingsModel) adjustVolLimit(step int) {
	v := m.volLimit + step
	if v < 5 {
		v = 5
	}
	if v > 100 {
		v = 100
	}
	m.volLimit = v
}

// applyActiveTab commits the change for the currently visible tab, if any.
func (m *SettingsModel) applyActiveTab() tea.Cmd {
	id := settingsTabs[m.activeTab].id
	switch id {
	case "tab.language":
		langs := state.AllLanguages()
		state.Current.Language = langs[m.langIdx]
		_ = state.Current.SaveConfig()
	case "tab.theme":
		if m.themeIdx >= len(themeNames) {
			// Custom row: open the hex editor.
			m.editingCustom = true
			m.customErr = ""
			if hex, ok := ui.ParseHexColor(state.Current.Theme); ok {
				m.customHex.SetValue(strings.TrimPrefix(hex, "#"))
			} else {
				m.customHex.SetValue("")
			}
			m.customHex.Focus()
			return nil
		}
		theme := themeNames[m.themeIdx]
		state.Current.Theme = theme
		_ = state.Current.SaveConfig()
		ui.ApplyTheme(theme)
		return func() tea.Msg { return ThemeChangedMsg{} }
	case "tab.sound":
		if len(m.soundDevices) > 0 && len(m.soundDevices) > m.soundSel {
			state.Current.SoundOutputDevice = m.soundDevices[m.soundSel].Name
		} else {
			state.Current.SoundOutputDevice = ""
		}
		state.Current.SoundVolumeLimit = m.volLimit
		_ = state.Current.SaveConfig()
		// Best-effort: route the audio backend to the chosen sink on next init.
		if len(m.soundDevices) > m.soundSel && m.soundSel >= 0 {
			d := m.soundDevices[m.soundSel]
			for _, e := range audio.RoutingEnv(d.Name, d.Card) {
				kv := strings.SplitN(e, "=", 2)
				if len(kv) == 2 {
					_ = os.Setenv(kv[0], kv[1])
				}
			}
		}
	case "tab.extras":
		names := ui.SpectrumPaletteNames()
		if m.spectrumIdx >= len(names) {
			// Auto-cycle row: Off -> 500ms -> 333ms -> Off.
			switch state.Current.SpectrumAutoCycleMs {
			case 0:
				state.Current.SpectrumAutoCycleMs = 500
			case 500:
				state.Current.SpectrumAutoCycleMs = 333
			default:
				state.Current.SpectrumAutoCycleMs = 0
				// Off: put the selected palette back to its original order.
				ui.SetSpectrumPalette(state.Current.SpectrumPalette)
			}
			_ = state.Current.SaveConfig()
			m.spectrumIdx = len(names)
			return nil
		}
		if m.spectrumIdx >= 0 && m.spectrumIdx < len(names) {
			state.Current.SpectrumPalette = names[m.spectrumIdx]
			_ = state.Current.SaveConfig()
			ui.SetSpectrumPalette(names[m.spectrumIdx])
		}
	}
	return nil
}

// cycleTab is invoked by the MainModel F3 handler: advance to the next tab,
// wrapping back to the first.
func (m *SettingsModel) cycleTab() {
	m.activeTab = (m.activeTab + 1) % len(settingsTabs)
	m.scroll = 0 // reset long-text scroll when switching tabs
	m.editingCustom = false
	m.customHex.Blur()
	m.customErr = ""
}

// cycleFocus exists for MainModel F1 player-bar focus cycling compatibility.
func (m *SettingsModel) cycleFocus() bool {
	m.focus = -1
	return true
}

// Button padding. Vertical padding is left at 0 so each button is a tidy
// single-line label; horizontal padding widens every button equally.
const (
	settingsBtnPadV = 0
	settingsBtnPadH = 4
)

func (m *SettingsModel) View() string {
	if m.width <= 0 {
		m.width = 120
	}
	if m.height <= 0 {
		m.height = 40
	}

	// Left panel = 35% of available width, right panel = remaining 65%.
	// Both scale automatically with the terminal width.
	totalW := m.width - 2
	if totalW < 40 {
		totalW = 40
	}
	leftW := totalW * 35 / 100
	gap := 3
	rightW := totalW - leftW - gap

	// Content height below the title line.
	contentH := m.height - 4
	if contentH < 10 {
		contentH = 10
	}

	title := ui.SectionTitleStyle.Render(" " + Tr("settings.title") + " ")

	leftPanel := m.renderLeftPanel(leftW, contentH)
	rightPanel := m.renderRightPanel(rightW, contentH)

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(leftW).Align(lipgloss.Center).Render(leftPanel),
		strings.Repeat(" ", gap),
		rightPanel,
	)

	return lipgloss.JoinVertical(lipgloss.Left, title, "", body)
}

// renderLeftPanel builds the vertically centered list of tab buttons.
// Buttons are spread evenly: equal gaps above, below, and between every button.
func (m *SettingsModel) renderLeftPanel(width int, height int) string {
	// Widest label across ALL 11 languages — every button pads to this, so
	// switching language never moves or resizes the buttons.
	maxW := 0
	for i := range settingsTabs {
		for _, l := range state.AllLanguages() {
			if w := lipgloss.Width(TrFor(l, settingsTabs[i].id)); w > maxW {
				maxW = w
			}
		}
	}
	innerW := maxW + settingsBtnPadH*2

	btnStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary)

	activeBtnStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Background(ui.ColorAccent).
		Foreground(ui.ColorBlack).
		Bold(true)

	var btns []string
	for i := range settingsTabs {
		label := centerPad(ui.FitLabel(m.tabLabel(i), innerW), innerW)
		if i == m.activeTab {
			btns = append(btns, activeBtnStyle.Render(label))
		} else {
			btns = append(btns, btnStyle.Render(label))
		}
	}

	// Reserve the hint at the very bottom; spread buttons in the remaining space.
	hint := ui.DimStyle.Render(Tr("settings.f3_hint"))
	hintH := lipgloss.Height(hint) + 2 // blank line above + the hint itself
	btnRegionH := height - hintH
	if btnRegionH < 1 {
		btnRegionH = 1
	}

	totalBtnH := 0
	for _, b := range btns {
		totalBtnH += lipgloss.Height(b)
	}
	// Equal gaps: one above the first, one between each pair, one below the last.
	numGaps := len(btns) + 1
	gapLines := (btnRegionH - totalBtnH) / numGaps
	if gapLines < 0 {
		gapLines = 0
	}
	blank := strings.Repeat("\n", gapLines)

	var b strings.Builder
	b.WriteString(blank)
	for i, btn := range btns {
		b.WriteString(btn)
		if i < len(btns)-1 {
			b.WriteString(blank)
		}
	}
	b.WriteString(blank)
	b.WriteString("\n")
	b.WriteString(hint)

	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		Render(b.String())
}

// renderRightPanel renders the content for the active tab inside its own border.
// The border uses the active theme's accent color so it reflects the selection.
func (m *SettingsModel) renderRightPanel(width int, height int) string {
	var content string
	switch settingsTabs[m.activeTab].id {
	case "tab.theme":
		content = m.renderThemeTab(width)
	case "tab.language":
		content = m.renderLangTab(width)
	case "tab.sound":
		content = m.renderSoundTab(width)
	case "tab.policies":
		content = m.renderScrollableTab(width, height, Tr("tab.policies"), policiesContent())
	case "tab.extras":
		content = m.renderExtrasTab(width, height)
	case "tab.about":
		content = m.renderScrollableTab(width, height, Tr("tab.about"), aboutContent())
	default:
		content = m.renderPlaceholder(width)
	}

	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorAccent).
		Width(width).
		Height(height).
		Align(lipgloss.Left, lipgloss.Top).
		Padding(0, 1)

	return style.Render(content)
}

func (m *SettingsModel) renderThemeTab(width int) string {
	var lines []string
	lines = append(lines, ui.SectionTitleStyle.Render(" "+Tr("tab.theme")+" "))
	lines = append(lines, "")
	// Presets render as a grid (navigation stays linear: every arrow key
	// steps to the next/previous theme in order).
	cols := 2
	if width >= 100 {
		cols = 3
	}
	cellW := 0
	for _, n := range themeNames {
		if w := ui.TextWidth("###  "+n) + 2; w > cellW {
			cellW = w
		}
	}
	for r := 0; r < len(themeNames); r += cols {
		cells := make([]string, 0, cols)
		for c := 0; c < cols; c++ {
			i := r + c
			if i >= len(themeNames) {
				cells = append(cells, strings.Repeat(" ", cellW))
				continue
			}
			n := themeNames[i]
			colorSample := lipgloss.NewStyle().Foreground(lipgloss.Color(ui.ThemeColors[n])).Render("###")
			cell := "  " + colorSample + "  " + n
			if m.rightFocused && i == m.themeIdx {
				cell = ui.AccentStyle.Bold(true).Render("> ") + colorSample + "  " + ui.WhiteStyle.Bold(true).Render(n)
			}
			cells = append(cells, ui.FitPad(cell, cellW))
		}
		lines = append(lines, strings.Join(cells, "  "))
	}
	// Trailing Custom row: current custom hex as its sample.
	customHex, isCustom := ui.ParseHexColor(state.Current.Theme)
	customSample := ui.DimStyle.Render("###")
	customName := Tr("theme.custom")
	if isCustom {
		customSample = lipgloss.NewStyle().Foreground(lipgloss.Color(customHex)).Render("###")
		customName += "  " + ui.DimStyle.Render(customHex)
	}
	customLine := "  " + customSample + "  " + customName
	if m.rightFocused && m.themeIdx == len(themeNames) {
		customLine = ui.AccentStyle.Bold(true).Render("> ") + customSample + "  " + ui.WhiteStyle.Bold(true).Render(customName)
	}
	lines = append(lines, customLine)
	if m.editingCustom {
		lines = append(lines, "", "  "+m.customHex.View())
		lines = append(lines, ui.DimStyle.Render("  "+Tr("theme.hex_hint")))
		if m.customErr != "" {
			lines = append(lines, ui.ErrorStyle.Render("  "+m.customErr))
		}
	} else {
		lines = append(lines, "")
		lines = append(lines, ui.DimStyle.Render("  "+Tr("settings.select_hint")))
	}
	return strings.Join(lines, "\n")
}

// renderExtrasTab shows the spectrum palette picker, the auto-cycle speed row
// and — under them — the shortcut/usage reference from extrasContent. The
// reference block scrolls with PgUp/PgDn (m.scroll) and is clipped to the panel.
func (m *SettingsModel) renderExtrasTab(width, height int) string {
	names := ui.SpectrumPaletteNames()
	previewW := width - 16
	if previewW < 8 {
		previewW = 8
	}
	innerW := width - 6
	if innerW < 24 {
		innerW = 24
	}
	innerH := height - 4
	if innerH < 8 {
		innerH = 8
	}

	head := []string{
		ui.SectionTitleStyle.Render(" " + Tr("tab.extras") + " "),
		"",
	}
	for i, n := range names {
		preview := ui.PalettePreview(n, previewW)
		line := "  " + preview + "  " + n
		if m.rightFocused && i == m.spectrumIdx {
			line = ui.AccentStyle.Bold(true).Render("> ") + preview + "  " + ui.WhiteStyle.Bold(true).Render(n)
		}
		head = append(head, line)
	}

	// Auto-cycle speed row, directly below the palette list.
	cycle := Tr("extras.cycle_off")
	switch state.Current.SpectrumAutoCycleMs {
	case 500:
		cycle = Tr("extras.cycle_500")
	case 333:
		cycle = Tr("extras.cycle_333")
	}
	cycleLabel := Tr("extras.auto_cycle") + ": " + cycle
	cycleRow := "  " + cycleLabel
	if m.rightFocused && m.spectrumIdx >= len(names) {
		cycleRow = ui.AccentStyle.Bold(true).Render("> ") +
			ui.WhiteStyle.Bold(true).Render(cycleLabel)
	}
	head = append(head, "", cycleRow, ui.DimStyle.Render("  "+Tr("extras.cycle_hint")),
		"", ui.DimStyle.Render("  "+Tr("settings.select_hint")))
	head = append(head, "")

	usageLines := wrapText(extrasContent(), innerW)
	avail := innerH - len(head) - 1
	if avail < 4 {
		avail = 4
	}
	if m.scroll > len(usageLines)-avail {
		m.scroll = len(usageLines) - avail
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	end := m.scroll + avail
	if end > len(usageLines) {
		end = len(usageLines)
	}
	lines := append([]string{}, head...)
	lines = append(lines, usageLines[m.scroll:end]...)
	if len(usageLines) > avail {
		lines = append(lines, ui.DimStyle.Render("  "+Tr("extras.scroll_hint")))
	}
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

func (m *SettingsModel) renderLangTab(width int) string {
	langs := state.AllLanguages()
	var items []string
	for i, l := range langs {
		// Native name always visible; localized name in parens when different.
		native := state.LanguageEndonym(l)
		localized := Tr("lang." + string(l))
		label := native
		if localized != native && localized != "lang."+string(l) {
			label = native + "  (" + localized + ")"
		}
		line := "  " + label
		if m.rightFocused && i == m.langIdx {
			line = ui.AccentStyle.Bold(true).Render("> ") + ui.WhiteStyle.Bold(true).Render(label)
		}
		items = append(items, line)
	}
	lines := []string{
		ui.SectionTitleStyle.Render(" " + Tr("tab.language") + " "),
		"",
	}
	lines = append(lines, items...)
	lines = append(lines, "")
	lines = append(lines, ui.DimStyle.Render("  "+Tr("settings.select_hint")))
	return strings.Join(lines, "\n")
}

// renderPlaceholder is shown for not-yet-implemented tabs.
func (m *SettingsModel) renderPlaceholder(width int) string {
	title := ui.SectionTitleStyle.Render(" " + m.tabLabel(m.activeTab) + " ")
	soon := ui.DimStyle.Render("  " + Tr("common.coming_soon"))
	return strings.Join([]string{title, "", soon}, "\n")
}

// renderSoundTab lists output devices with a Bluetooth/Wired badge and a
// volume-limit slider. Up/Down move the selection; Left/Right change the
// limit (5-100%). Enter saves.
func (m *SettingsModel) renderSoundTab(width int) string {
	title := ui.SectionTitleStyle.Render(" " + Tr("tab.sound") + " ")
	lines := []string{title, ""}

	if len(m.soundDevices) == 0 {
		lines = append(lines, ui.DimStyle.Render("  "+Tr("sound.no_devices")))
	} else {
		// Badges share one fixed width (longest translation of either label
		// across all 11 languages) so device rows never shift.
		badgeW := maxTrWidth("settings.badge_wired")
		if w := maxTrWidth("settings.badge_bt"); w > badgeW {
			badgeW = w
		}
		for i, d := range m.soundDevices {
			badge := ui.DimStyle.Render(ui.FitPad(Tr("settings.badge_wired"), badgeW))
			if d.Type == "bluetooth" {
				badge = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(ui.FitPad(Tr("settings.badge_bt"), badgeW))
			}
			line := "  " + badge + "  " + d.Description
			if m.rightFocused && i == m.soundSel {
				line = ui.AccentStyle.Bold(true).Render("> ") + badge + "  " + ui.WhiteStyle.Bold(true).Render(d.Description)
			}
			lines = append(lines, line)
		}
	}

	// Volume limit slider
	lines = append(lines, "", ui.SectionTitleStyle.Render(" "+Tr("sound.volume_limit")+" "))
	bar := m.volumeBar(m.volLimit, width-6)
	lines = append(lines, "  "+bar+"  "+ui.WhiteStyle.Render(fmt.Sprintf("%d%%", m.volLimit)))
	lines = append(lines, "", ui.DimStyle.Render("  "+Tr("sound.hint")))

	return strings.Join(lines, "\n")
}

// volumeBar renders a simple [=====     ] progress bar of the given percent.
func (m *SettingsModel) volumeBar(pct, w int) string {
	if w < 10 {
		w = 10
	}
	filled := int(float64(pct) / 100.0 * float64(w))
	if filled > w {
		filled = w
	}
	if filled < 0 {
		filled = 0
	}
	return "[" + strings.Repeat("=", filled) + strings.Repeat(" ", w-filled) + "]"
}

// handleScrollKey intercepts ↑/↓/PgUp/PgDn for the long-text tabs so the user
// can read Policies/About without leaving the Settings view.
func (m *SettingsModel) handleScrollKey(s string) (bool, tea.Cmd) {
	id := settingsTabs[m.activeTab].id
	switch id {
	case "tab.policies", "tab.about":
		// Full-page text: ↑/↓ scroll line by line.
	case "tab.extras":
		// ↑/↓ belong to the palette list here, so only Page keys scroll.
		switch s {
		case "pgup":
			m.scrollBy(-10)
		case "pgdown":
			m.scrollBy(10)
		default:
			return false, nil
		}
		return true, nil
	default:
		return false, nil
	}
	switch s {
	case "up", "k":
		m.scrollBy(-1)
	case "down", "j":
		m.scrollBy(1)
	case "pgup":
		m.scrollBy(-10)
	case "pgdown":
		m.scrollBy(10)
	default:
		return false, nil
	}
	return true, nil
}

// scrollBy moves the long-text view; the renderer clamps to valid range.
func (m *SettingsModel) scrollBy(d int) {
	m.scroll += d
	if m.scroll < 0 {
		m.scroll = 0
	}
}

// wrapText hard-wraps text to width w at word boundaries.
func wrapText(text string, w int) []string {
	if w < 10 {
		w = 10
	}
	var out []string
	for _, para := range strings.Split(text, "\n") {
		if para == "" {
			out = append(out, "")
			continue
		}
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		line := ""
		for _, word := range words {
			if line == "" {
				line = word
			} else if lipgloss.Width(line)+1+lipgloss.Width(word) <= w {
				line += " " + word
			} else {
				out = append(out, line)
				line = word
			}
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// renderScrollableTab renders a title plus word-wrapped, vertically scrollable
// text with a simple scrollbar on the right edge.
func (m *SettingsModel) renderScrollableTab(width, height int, title, raw string) string {
	innerW := width - 6 // borders + padding
	innerH := height - 4
	if innerH < 4 {
		innerH = 4
	}
	lines := wrapText(raw, innerW)
	maxScroll := len(lines) - innerH
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.scroll > maxScroll {
		m.scroll = maxScroll
	}
	if m.scroll < 0 {
		m.scroll = 0
	}
	start := m.scroll
	end := start + innerH
	if end > len(lines) {
		end = len(lines)
	}
	visible := lines[start:end]

	body := strings.Join(visible, "\n")
	if len(visible) < innerH {
		body += strings.Repeat("\n", innerH-len(visible))
	}

	// Scrollbar
	var sb strings.Builder
	for i := 0; i < innerH; i++ {
		if maxScroll == 0 {
			sb.WriteString(ui.FaintStyle.Render("│"))
		} else if i == int(float64(m.scroll)/float64(maxScroll)*float64(innerH-1)) {
			sb.WriteString(ui.AccentStyle.Render("█"))
		} else {
			sb.WriteString(ui.FaintStyle.Render("│"))
		}
	}
	content := lipgloss.JoinHorizontal(lipgloss.Top,
		ui.SectionTitleStyle.Render(" "+title+" ")+"\n"+body,
		" ", sb.String())
	return content
}

// centerPad pads s with spaces so its display width equals targetW, centering
// the text. Used to make button borders all line up at an identical width.
func centerPad(s string, targetW int) string {
	w := lipgloss.Width(s)
	if w >= targetW {
		return s
	}
	total := targetW - w
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
