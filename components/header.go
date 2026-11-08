package components

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"MusicLe/state"
	"MusicLe/ui"
)

func RenderHeader(width int, activeView string) string {
	divStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Padding(0, 2)

	logoText := ui.LogoStyle.Render("Music") + ui.LogoAccentStyle.Render("Le")
	logoDiv := divStyle.Render(logoText)

	netColor := ui.ColorAccent
	if !state.Current.NetworkOnline {
		netColor = lipgloss.Color("#666666")
	}
	netIndicator := lipgloss.NewStyle().Foreground(netColor).Render("o")
	clock := time.Now().Format("15:04")
	lang := langBadge()
	statusDiv := divStyle.Render(fmt.Sprintf("%s %s %s", netIndicator, clock, lang))

	logoW := lipgloss.Width(logoDiv)
	statusW := lipgloss.Width(statusDiv)
	avail := width - 2

	// Each tab box keeps the width of its longest label across all 11
	// languages, so switching language never moves or wraps a tab.
	// Narrow terminals degrade in stages: first labels shrink with an
	// ellipsis, then the logo is hidden, always keeping a single row.
	showLogo := true
	tabContentW := make(map[string]int, len(navTabIDs))
	tabTotal := 0
	shrink := func(fixed int) {
		budget := (avail - fixed - 2*len(navTabIDs)) / len(navTabIDs)
		budget -= navTabPadH * 2
		if budget < 2 {
			budget = 2
		}
		tabTotal = 0
		for _, id := range navTabIDs {
			w := NavTabWidth(id)
			if w > budget+navTabPadH*2 {
				w = budget + navTabPadH*2
			}
			tabContentW[id] = w
			tabTotal += w + 2 // +2 for the rounded borders
		}
	}
	for _, id := range navTabIDs {
		w := NavTabWidth(id)
		tabContentW[id] = w
		tabTotal += w + 2
	}
	if fixed := logoW + statusW + 4; fixed+tabTotal > avail {
		shrink(fixed)
	}
	if fixed := logoW + statusW + 4; showLogo && fixed+tabTotal > avail {
		// Still overflowing: hide the logo and re-fit the tabs.
		showLogo = false
		logoW = 0
		shrink(statusW + 4)
	}

	tabBase := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, navTabPadH).
		Align(lipgloss.Center)

	var tabs []string
	for _, id := range navTabIDs {
		label := ui.FitPadCenter(NavLabel(id), tabContentW[id]-navTabPadH*2)
		st := tabBase.Width(tabContentW[id])
		if activeView == id {
			st = st.BorderForeground(ui.ColorAccent).
				Background(ui.ColorAccent).
				Foreground(ui.ColorBlack).
				Bold(true)
		} else {
			st = st.BorderForeground(ui.ColorPrimary)
		}
		tabs = append(tabs, st.Render(label))
	}
	tabsJoined := lipgloss.JoinHorizontal(lipgloss.Center, tabs...)

	tabsW := lipgloss.Width(tabsJoined)
	remaining := avail - logoW - tabsW - statusW - 4
	if remaining < 0 {
		remaining = 0
	}
	left := remaining / 2
	right := remaining - left

	parts := []string{"  "}
	if showLogo {
		parts = append(parts, logoDiv)
	}
	parts = append(parts,
		strings.Repeat(" ", left),
		tabsJoined,
		strings.Repeat(" ", right),
		statusDiv,
		"  ",
	)
	row := lipgloss.JoinHorizontal(lipgloss.Center, parts...)

	outer := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.ColorPrimary).
		Width(width - 2)

	return outer.Render(row)
}
