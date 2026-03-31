package component

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"
)

const multiStringMaxVisible = 7

// MultiStringInput is a comma-separated free-text input with fuzzy autocomplete
// from a fixed list of suggestions (e.g. keywords).
type MultiStringInput struct {
	input    textinput.Model
	items    []string
	matches  []fuzzy.Match
	cursor   int
	open     bool
	focused  bool
}

// NewMultiStringInput creates a new input with the given suggestions.
func NewMultiStringInput(placeholder string, items []string) MultiStringInput {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 512

	m := MultiStringInput{
		input: ti,
		items: items,
	}
	return m
}

// Value returns the raw comma-separated input.
func (m MultiStringInput) Value() string {
	return strings.TrimSpace(m.input.Value())
}

// Values returns the parsed, trimmed, non-empty entries.
func (m MultiStringInput) Values() []string {
	var out []string
	for _, p := range strings.Split(m.input.Value(), ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// SetValue sets the input value directly.
func (m *MultiStringInput) SetValue(v string) {
	m.input.SetValue(v)
}

// Focus activates the input.
func (m *MultiStringInput) Focus() {
	m.focused = true
	m.input.Focus()
	m.filterCurrent()
	m.open = len(m.matches) > 0
}

// Blur deactivates the input.
func (m *MultiStringInput) Blur() {
	m.focused = false
	m.input.Blur()
	m.open = false
}

// IsOpen returns whether the dropdown is currently visible.
func (m MultiStringInput) IsOpen() bool {
	return m.open
}

func (m MultiStringInput) Init() tea.Cmd { return nil }

func (m MultiStringInput) Update(msg tea.Msg) (MultiStringInput, tea.Cmd) {
	if !m.focused {
		return m, nil
	}

	if len(m.items) == 0 {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyUp:
			if m.open && m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown:
			if m.open && m.cursor < len(m.matches)-1 {
				m.cursor++
			}
			return m, nil

		case tea.KeyEnter:
			if m.open && len(m.matches) > 0 {
				m.selectMatch(m.cursor)
				return m, nil
			}

		case tea.KeyEscape:
			if m.open {
				m.open = false
				return m, nil
			}
		}
	}

	prevValue := m.input.Value()
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	if m.input.Value() != prevValue {
		m.filterCurrent()
		m.open = len(m.matches) > 0
	}

	return m, cmd
}

func (m *MultiStringInput) selectMatch(idx int) {
	if idx < 0 || idx >= len(m.matches) {
		return
	}
	value := m.items[m.matches[idx].Index]

	raw := m.input.Value()
	prefix := ""
	if lastComma := strings.LastIndex(raw, ","); lastComma >= 0 {
		prefix = raw[:lastComma+1] + " "
	}

	m.input.SetValue(prefix + value + ", ")
	m.input.CursorEnd()
	m.filterCurrent()
	m.open = len(m.matches) > 0
}

// currentFragment returns the text after the last comma.
func (m *MultiStringInput) currentFragment() string {
	raw := m.input.Value()
	if idx := strings.LastIndex(raw, ","); idx >= 0 {
		return strings.TrimSpace(raw[idx+1:])
	}
	return strings.TrimSpace(raw)
}

// alreadyEntered returns values typed before the current segment.
func (m *MultiStringInput) alreadyEntered() map[string]bool {
	entered := make(map[string]bool)
	raw := m.input.Value()
	parts := strings.Split(raw, ",")
	if len(parts) > 1 {
		for _, p := range parts[:len(parts)-1] {
			if t := strings.TrimSpace(p); t != "" {
				entered[strings.ToLower(t)] = true
			}
		}
	}
	return entered
}

func (m *MultiStringInput) filterCurrent() {
	fragment := m.currentFragment()
	already := m.alreadyEntered()

	var filtered []string
	var indexMap []int
	for i, item := range m.items {
		if already[strings.ToLower(item)] {
			continue
		}
		filtered = append(filtered, item)
		indexMap = append(indexMap, i)
	}

	if fragment == "" {
		m.matches = make([]fuzzy.Match, len(filtered))
		for i := range filtered {
			m.matches[i] = fuzzy.Match{Str: filtered[i], Index: indexMap[i]}
		}
	} else {
		raw := fuzzy.Find(fragment, filtered)
		m.matches = make([]fuzzy.Match, len(raw))
		for i, match := range raw {
			m.matches[i] = fuzzy.Match{
				Str:            match.Str,
				Index:          indexMap[match.Index],
				MatchedIndexes: match.MatchedIndexes,
				Score:          match.Score,
			}
		}
	}
	m.cursor = 0
}

// View renders the input and dropdown.
func (m MultiStringInput) View() string {
	var b strings.Builder

	if !m.focused {
		val := strings.TrimRight(strings.TrimSpace(m.input.Value()), ",")
		val = strings.TrimSpace(val)
		if val == "" {
			b.WriteString(bidEmptyStyle.Render("none"))
		} else {
			b.WriteString(bidDisplayStyle.Render(val))
		}
		return b.String()
	}

	b.WriteString(m.input.View())

	if !m.open || len(m.matches) == 0 {
		return b.String()
	}

	b.WriteString("\n")

	start, end := visibleWindow(m.cursor, len(m.matches), multiStringMaxVisible)
	total := len(m.matches)

	if start > 0 {
		b.WriteString(scrollHintStyle.Render("  ↑ " + fmt.Sprintf("%d more", start)))
		b.WriteString("\n")
	}

	for i := start; i < end; i++ {
		match := m.matches[i]
		if i == m.cursor {
			b.WriteString(cursorStyle.Render("▸ ") + selectedItemStyle.Render(highlightMatch(match)))
		} else {
			b.WriteString(itemStyle.Render("  " + highlightMatch(match)))
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
