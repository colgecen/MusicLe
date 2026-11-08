package main

import (
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"MusicLe/state"
	"MusicLe/ui"
)

func renderLangModal(lang state.Language) string {
	langs := state.AllLanguages()
	var rows []string
	for _, l := range langs {
		name := state.LanguageEndonym(l)
		if l == lang {
			rows = append(rows, ui.AccentStyle.Render("> "+name))
		} else {
			rows = append(rows, "  "+name)
		}
	}
	langOpts := strings.Join(rows, "\n")

	// Title/hint in the currently highlighted language so the user sees
	// immediate feedback while cycling.
	saved := state.Current.Language
	state.Current.Language = lang
	hintTitle := Tr("setup.lang_title")
	hintKeys := Tr("setup.hint")
	welcome := Tr("setup.welcome")
	state.Current.Language = saved

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		renderLogo(),
		"",
		ui.WhiteStyle.Bold(true).Render(hintTitle),
		"",
		"  "+langOpts,
		"",
		ui.DimStyle.Render(hintKeys),
	)

	title := ui.WhiteStyle.Render("  " + ui.LogoStyle.Render("Music") + ui.LogoAccentStyle.Render("Le") + "  " + welcome)
	box := ui.AccentBorderStyle.
		Width(46).
		Render(title + "\n" + content)

	return lipgloss.Place(50, 16, lipgloss.Center, lipgloss.Center, box)
}

func placeOverlay(full, overlay string, width int) string {
	lines := strings.Split(full, "\n")
	totalH := len(lines)
	overlayH := lipgloss.Height(overlay)
	overlayW := lipgloss.Width(overlay)
	topPad := (totalH - overlayH) / 2
	leftPad := (width - overlayW) / 2
	if topPad < 0 {
		topPad = 0
	}
	if leftPad < 0 {
		leftPad = 0
	}
	overlayLines := strings.Split(overlay, "\n")
	var result []string
	for i, line := range lines {
		if i >= topPad && i < topPad+overlayH {
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

func initializeDefaults(lang state.Language) tea.Cmd {
	return func() tea.Msg {
		state.Current.Language = lang

		homeDir, err := os.UserHomeDir()
		if err != nil {
			homeDir = "."
		}
		rootDir := filepath.Join(homeDir, "Music", "MusicLe")
		if err := state.Current.InitializeBaseDirs(rootDir); err != nil {
			return errorMsg(err.Error())
		}
		if err := state.Current.CreateProfileStructure("default", "Default", "", lang); err != nil {
			return errorMsg(err.Error())
		}
		if err := state.Current.CreatePlaylistStructure("default", "my-playlist", "My Playlist", "", ""); err != nil {
			return errorMsg(err.Error())
		}
		if err := state.Current.SaveConfig(); err != nil {
			return errorMsg(err.Error())
		}
		_ = state.Current.ScanProfiles()
		if len(state.Current.Profiles) > 0 {
			state.Current.CurrentProfile = &state.Current.Profiles[0]
			if len(state.Current.CurrentProfile.Playlists) > 0 {
				state.Current.CurrentPlaylist = &state.Current.CurrentProfile.Playlists[0]
			}
		}
		state.Current.IsFirstLaunch = false
		return setupDoneMsg{}
	}
}
