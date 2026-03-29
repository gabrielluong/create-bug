package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sahilm/fuzzy"
)

const maxVisible = 7

// FuzzySelect is a text input with a fuzzy-filtered dropdown.
// If items is empty, it behaves as a plain free-text input.
type FuzzySelect struct {
	input        textinput.Model
	items        []string
	matches      []fuzzy.Match
	cursor       int
	selected     string
	prevSelected string
	open         bool
	focused      bool
}

// NewFuzzySelect creates a new fuzzy select with the given items.
func NewFuzzySelect(placeholder string, items []string) FuzzySelect {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 128

	fs := FuzzySelect{
		input: ti,
		items: items,
	}
	fs.filter()
	return fs
}

// Value returns the confirmed selection, or the raw input if no selection.
func (f FuzzySelect) Value() string {
	if f.selected != "" {
		return f.selected
	}
	return strings.TrimSpace(f.input.Value())
}

// SetValue sets the input and selected value directly.
func (f *FuzzySelect) SetValue(v string) {
	f.input.SetValue(v)
	f.selected = v
	f.open = false
}

// Focus activates the input.
func (f *FuzzySelect) Focus() {
	f.focused = true
	f.input.Focus()
	// Clear the input so the full list is visible for browsing.
	f.prevSelected = f.selected
	f.selected = ""
	f.input.SetValue("")
	f.open = len(f.items) > 0
	f.filter()
}

// Blur deactivates the input. Auto-confirms the best match if the user
// typed something, or restores the previous selection if input is empty.
func (f *FuzzySelect) Blur() {
	f.focused = false
	f.input.Blur()
	if f.selected == "" {
		typed := strings.TrimSpace(f.input.Value())
		if typed != "" && len(f.matches) > 0 {
			f.selected = f.matches[f.cursor].Str
		} else if typed == "" && f.prevSelected != "" {
			f.selected = f.prevSelected
		}
	}
	f.input.SetValue(f.selected)
	f.open = false
	f.prevSelected = ""
}

// Focused returns whether the selector is focused.
func (f FuzzySelect) Focused() bool {
	return f.focused
}

// IsOpen returns whether the dropdown is currently visible.
func (f FuzzySelect) IsOpen() bool {
	return f.open
}

func (f FuzzySelect) Init() tea.Cmd {
	return nil
}

func (f FuzzySelect) Update(msg tea.Msg) (FuzzySelect, tea.Cmd) {
	if !f.focused {
		return f, nil
	}

	// Free-text mode when no items are configured.
	if len(f.items) == 0 {
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return f, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyUp:
			if f.open && f.cursor > 0 {
				f.cursor--
			}
			return f, nil

		case tea.KeyDown:
			if f.open && f.cursor < len(f.matches)-1 {
				f.cursor++
			}
			return f, nil

		case tea.KeyEnter:
			if f.open && len(f.matches) > 0 {
				f.selected = f.matches[f.cursor].Str
				f.input.SetValue(f.selected)
				f.open = false
				// Move cursor to end of input.
				f.input.CursorEnd()
			}
			return f, nil

		case tea.KeyEscape:
			if f.open {
				f.open = false
				return f, nil
			}
		}
	}

	prevValue := f.input.Value()
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)

	// Re-filter if input changed.
	if f.input.Value() != prevValue {
		f.selected = ""
		f.open = true
		f.filter()
	}

	return f, cmd
}

func (f *FuzzySelect) filter() {
	query := strings.TrimSpace(f.input.Value())
	if query == "" {
		// Show all items when empty.
		f.matches = make([]fuzzy.Match, len(f.items))
		for i, item := range f.items {
			f.matches[i] = fuzzy.Match{Str: item, Index: i}
		}
	} else {
		f.matches = fuzzy.Find(query, f.items)
	}
	f.cursor = 0
}

// View renders the input and dropdown.
func (f FuzzySelect) View() string {
	var b strings.Builder

	if !f.focused && f.selected != "" {
		// Show a compact read-only display when blurred.
		b.WriteString(selectedDisplayStyle.Render(f.selected))
		return b.String()
	}

	b.WriteString(f.input.View())

	if !f.open || len(f.matches) == 0 {
		return b.String()
	}

	b.WriteString("\n")

	// Scrollbar hint.
	start, end := visibleWindow(f.cursor, len(f.matches), maxVisible)
	total := len(f.matches)

	if start > 0 {
		b.WriteString(scrollHintStyle.Render("  ↑ " + fmt.Sprintf("%d more", start)))
		b.WriteString("\n")
	}

	for i := start; i < end; i++ {
		m := f.matches[i]
		if i == f.cursor {
			b.WriteString(cursorStyle.Render("▸ ") + selectedItemStyle.Render(highlightMatch(m)))
		} else {
			b.WriteString(itemStyle.Render("  " + highlightMatch(m)))
		}
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	if end < total {
		b.WriteString("\n")
		b.WriteString(scrollHintStyle.Render("  ↓ " + fmt.Sprintf("%d more", total-end)))
	}

	return b.String()
}

// visibleWindow computes the start/end indices for a scrolling window.
func visibleWindow(cursor, total, max int) (int, int) {
	if total <= max {
		return 0, total
	}
	half := max / 2
	start := cursor - half
	if start < 0 {
		start = 0
	}
	end := start + max
	if end > total {
		end = total
		start = end - max
	}
	return start, end
}

// highlightMatch renders a match with matched characters highlighted.
func highlightMatch(m fuzzy.Match) string {
	if len(m.MatchedIndexes) == 0 {
		return m.Str
	}

	matched := make(map[int]bool, len(m.MatchedIndexes))
	for _, idx := range m.MatchedIndexes {
		matched[idx] = true
	}

	var b strings.Builder
	for i, ch := range m.Str {
		if matched[i] {
			b.WriteString(matchHighlightStyle.Render(string(ch)))
		} else {
			b.WriteRune(ch)
		}
	}
	return b.String()
}

var (
	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#E2E8F0"))

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B"))

	selectedDisplayStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#C4B5FD"))

	scrollHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#475569")).
			Italic(true)

	matchHighlightStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#D946EF"))
)
