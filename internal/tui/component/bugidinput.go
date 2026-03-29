package component

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gabrielluong/create-bug/internal/history"
	"github.com/sahilm/fuzzy"
)

const bugIDMaxVisible = 10

// historyItem is a display string for fuzzy matching with a reference to the entry.
type historyItem struct {
	display string
	entry   history.Entry
}

// BugIDInput is a comma-separated bug ID input with history autocomplete.
type BugIDInput struct {
	input   textinput.Model
	history []historyItem
	matches []fuzzy.Match
	cursor  int
	open    bool
	focused bool
}

// NewBugIDInput creates a new bug ID input populated with history entries.
func NewBugIDInput(placeholder string, entries []history.Entry) BugIDInput {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256

	// Sort entries: meta bugs first, then by recency (newest first).
	sorted := make([]history.Entry, len(entries))
	copy(sorted, entries)
	sort.SliceStable(sorted, func(i, j int) bool {
		mi := isMetaBug(sorted[i].Summary)
		mj := isMetaBug(sorted[j].Summary)
		if mi != mj {
			return mi
		}
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})

	items := make([]historyItem, len(sorted))
	for i, e := range sorted {
		items[i] = historyItem{
			display: fmt.Sprintf("%d — %s [%s :: %s]", e.ID, e.Summary, e.Product, e.Component),
			entry:   e,
		}
	}

	return BugIDInput{
		input:   ti,
		history: items,
	}
}

// Value returns the raw comma-separated input.
func (b BugIDInput) Value() string {
	return strings.TrimSpace(b.input.Value())
}

// IDs parses the input into integer bug IDs, skipping invalid entries.
func (b BugIDInput) IDs() []int {
	raw := b.input.Value()
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	var ids []int
	for _, part := range strings.Split(raw, ",") {
		t := strings.TrimSpace(part)
		if t == "" {
			continue
		}
		if n, err := strconv.Atoi(t); err == nil {
			ids = append(ids, n)
		}
	}
	return ids
}

// Focus activates the input.
func (b *BugIDInput) Focus() {
	b.focused = true
	b.input.Focus()
	b.filterCurrent()
	b.open = len(b.matches) > 0
}

// IsOpen returns whether the dropdown is currently visible.
func (b BugIDInput) IsOpen() bool {
	return b.open
}

// Blur deactivates the input.
func (b *BugIDInput) Blur() {
	b.focused = false
	b.input.Blur()
	b.open = false
}

func (b BugIDInput) Init() tea.Cmd {
	return nil
}

func (b BugIDInput) Update(msg tea.Msg) (BugIDInput, tea.Cmd) {
	if !b.focused {
		return b, nil
	}

	if len(b.history) == 0 {
		var cmd tea.Cmd
		b.input, cmd = b.input.Update(msg)
		return b, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyUp:
			if b.open && b.cursor > 0 {
				b.cursor--
			}
			return b, nil

		case tea.KeyDown:
			if b.open && b.cursor < len(b.matches)-1 {
				b.cursor++
			}
			return b, nil

		case tea.KeyEnter:
			if b.open && len(b.matches) > 0 {
				b.selectMatch(b.cursor)
				return b, nil
			}

		case tea.KeyEscape:
			if b.open {
				b.open = false
				return b, nil
			}
		}
	}

	prevValue := b.input.Value()
	var cmd tea.Cmd
	b.input, cmd = b.input.Update(msg)

	if b.input.Value() != prevValue {
		b.filterCurrent()
		b.open = len(b.matches) > 0
	}

	return b, cmd
}

// selectMatch inserts the bug ID from the matched entry at the current cursor.
func (b *BugIDInput) selectMatch(idx int) {
	if idx < 0 || idx >= len(b.matches) {
		return
	}
	entry := b.history[b.matches[idx].Index].entry
	idStr := strconv.Itoa(entry.ID)

	raw := b.input.Value()
	prefix := ""
	if lastComma := strings.LastIndex(raw, ","); lastComma >= 0 {
		prefix = raw[:lastComma+1] + " "
	}

	b.input.SetValue(prefix + idStr + ", ")
	b.input.CursorEnd()
	b.filterCurrent()
	b.open = len(b.matches) > 0
}

// currentFragment returns the text after the last comma (the segment being typed).
func (b *BugIDInput) currentFragment() string {
	raw := b.input.Value()
	if idx := strings.LastIndex(raw, ","); idx >= 0 {
		return strings.TrimSpace(raw[idx+1:])
	}
	return strings.TrimSpace(raw)
}

// alreadyEntered returns IDs already typed before the current fragment.
func (b *BugIDInput) alreadyEntered() map[int]bool {
	entered := make(map[int]bool)
	raw := b.input.Value()
	parts := strings.Split(raw, ",")
	// Exclude the last segment (currently being typed).
	if len(parts) > 1 {
		for _, p := range parts[:len(parts)-1] {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
				entered[n] = true
			}
		}
	}
	return entered
}

// filterCurrent filters history entries against the current fragment,
// excluding IDs already entered.
func (b *BugIDInput) filterCurrent() {
	fragment := b.currentFragment()
	already := b.alreadyEntered()

	// Build the list of display strings for matching, excluding already-entered.
	var displays []string
	var indexMap []int // maps filtered index → b.history index
	for i, h := range b.history {
		if already[h.entry.ID] {
			continue
		}
		displays = append(displays, h.display)
		indexMap = append(indexMap, i)
	}

	if fragment == "" {
		// Show all remaining.
		b.matches = make([]fuzzy.Match, len(displays))
		for i := range displays {
			b.matches[i] = fuzzy.Match{Str: displays[i], Index: indexMap[i]}
		}
	} else {
		raw := fuzzy.Find(fragment, displays)
		b.matches = make([]fuzzy.Match, len(raw))
		for i, m := range raw {
			b.matches[i] = fuzzy.Match{
				Str:            m.Str,
				Index:          indexMap[m.Index],
				MatchedIndexes: m.MatchedIndexes,
				Score:          m.Score,
			}
		}
	}
	b.cursor = 0
}

// View renders the input and autocomplete dropdown.
func (b BugIDInput) View() string {
	var s strings.Builder

	if !b.focused {
		val := strings.TrimSpace(b.input.Value())
		val = strings.TrimRight(val, ",")
		val = strings.TrimSpace(val)
		if val == "" {
			s.WriteString(bidEmptyStyle.Render("none"))
		} else {
			s.WriteString(bidDisplayStyle.Render(val))
		}
		return s.String()
	}

	s.WriteString(b.input.View())

	if !b.open || len(b.matches) == 0 {
		return s.String()
	}

	s.WriteString("\n")

	start, end := visibleWindow(b.cursor, len(b.matches), bugIDMaxVisible)
	total := len(b.matches)

	if start > 0 {
		s.WriteString(scrollHintStyle.Render("  ↑ " + fmt.Sprintf("%d more", start)))
		s.WriteString("\n")
	}

	for i := start; i < end; i++ {
		m := b.matches[i]
		h := b.history[m.Index]
		if i == b.cursor {
			s.WriteString(cursorStyle.Render("▸ "))
			s.WriteString(bidIDStyle.Render(strconv.Itoa(h.entry.ID)))
			s.WriteString(bidSepStyle.Render(" — "))
			s.WriteString(bidSummaryStyle.Render(truncate(h.entry.Summary, 40)))
			s.WriteString(bidMetaStyle.Render(" ["+h.entry.Product+" :: "+h.entry.Component+"]"))
		} else {
			s.WriteString("  ")
			s.WriteString(bidIDDimStyle.Render(strconv.Itoa(h.entry.ID)))
			s.WriteString(bidSepStyle.Render(" — "))
			s.WriteString(bidSummaryDimStyle.Render(truncate(h.entry.Summary, 40)))
			s.WriteString(bidMetaDimStyle.Render(" ["+h.entry.Product+" :: "+h.entry.Component+"]"))
		}
		if i < end-1 {
			s.WriteString("\n")
		}
	}

	if end < total {
		s.WriteString("\n")
		s.WriteString(scrollHintStyle.Render("  ↓ " + fmt.Sprintf("%d more", total-end)))
	}

	return s.String()
}

// isMetaBug checks if a bug summary contains a meta tag like [meta], [Meta], [META], etc.
func isMetaBug(summary string) bool {
	lower := strings.ToLower(summary)
	return strings.Contains(lower, "[meta]")
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

var (
	bidDisplayStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C4B5FD"))

	bidEmptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569")).
			Italic(true)

	bidIDStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Bold(true)

	bidIDDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B"))

	bidSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#334155"))

	bidSummaryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0"))

	bidSummaryDimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94A3B8"))

	bidMetaStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B"))

	bidMetaDimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569"))
)
