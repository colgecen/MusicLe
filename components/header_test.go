package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"MusicLe/state"
	"MusicLe/ui"
)

// TestNavTabWidthCoversAllLanguages asserts every tab box fits the longest
// label among all 11 languages, so no tab can ever wrap or grow.
func TestNavTabWidthCoversAllLanguages(t *testing.T) {
	for _, id := range navTabIDs {
		labels, ok := navLabels[id]
		if !ok {
			t.Fatalf("tab %q missing from navLabels", id)
		}
		if len(labels) != len(state.AllLanguages()) {
			t.Errorf("tab %q: got %d labels, want %d (one per language)",
				id, len(labels), len(state.AllLanguages()))
		}
		budget := NavTabWidth(id) - navTabPadH*2
		for lang, s := range labels {
			if w := ui.TextWidth(s); w > budget {
				t.Errorf("tab %q lang %q: label width %d exceeds box %d (%q)",
					id, lang, w, budget, s)
			}
		}
	}
}

// TestHeaderStableAcrossLanguages renders the header in every language and
// asserts identical geometry: same height, same first-line width. Any wrapped
// tab would change the height; any resized box would change the width.
func TestHeaderStableAcrossLanguages(t *testing.T) {
	saved := state.Current.Language
	defer func() { state.Current.Language = saved }()

	const width = 160
	var wantH, wantW int
	for i, lang := range state.AllLanguages() {
		state.Current.Language = lang
		out := RenderHeader(width, "home")
		h := lipgloss.Height(out)
		first := strings.SplitN(out, "\n", 2)[0]
		w := lipgloss.Width(first)
		if i == 0 {
			wantH, wantW = h, w
			t.Logf("lang %q height=%d firstLineWidth=%d", lang, h, w)
			continue
		}
		if h != wantH {
			t.Errorf("lang %q: header height %d, want %d (a tab wrapped?)", lang, h, wantH)
		}
		if w != wantW {
			t.Errorf("lang %q: first-line width %d, want %d (a tab resized?)", lang, w, wantW)
		}
	}
}

// TestHeaderNarrowTerminal asserts the header degrades gracefully (ellipsis,
// no wrap explosion) on an 80-column terminal in every language.
func TestHeaderNarrowTerminal(t *testing.T) {
	saved := state.Current.Language
	defer func() { state.Current.Language = saved }()

	for _, lang := range state.AllLanguages() {
		state.Current.Language = lang
		out := RenderHeader(80, "downloads")
		if h := lipgloss.Height(out); h > 7 {
			t.Errorf("lang %q: narrow header height %d, want <= 7", lang, h)
		}
	}
}

// TestRenderAllLanguagesVisual logs the rendered header per language for
// manual eyeballing: go test ./components/ -run TestRenderAllLanguagesVisual -v
func TestRenderAllLanguagesVisual(t *testing.T) {
	saved := state.Current.Language
	defer func() { state.Current.Language = saved }()

	for _, lang := range state.AllLanguages() {
		state.Current.Language = lang
		t.Logf("=== %s (%s) ===\n%s", state.LanguageEndonym(lang), lang, RenderHeader(160, "downloads"))
	}
}
