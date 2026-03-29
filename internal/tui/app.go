package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gabrielluong/create-bug/internal/config"
	"github.com/gabrielluong/create-bug/internal/tui/form"
)

// Screen identifies the current TUI screen.
type Screen int

const (
	ScreenForm Screen = iota
	ScreenHistory
)

// AppModel is the root Bubbletea model.
type AppModel struct {
	cfg    *config.Config
	keys   KeyMap
	screen Screen
	form   form.Model
	width  int
	height int
}

// NewApp creates a new AppModel with the given config.
func NewApp(cfg *config.Config) AppModel {
	return AppModel{
		cfg:    cfg,
		keys:   DefaultKeyMap(),
		screen: ScreenForm,
		form:   form.New(cfg),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.form.SetSize(msg.Width, msg.Height)
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
	}

	switch m.screen {
	case ScreenForm:
		var cmd tea.Cmd
		m.form, cmd = m.form.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m AppModel) View() string {
	padH := 3
	if m.width < 60 {
		padH = 1
	} else if m.width < 80 {
		padH = 2
	}
	contentStyle := lipgloss.NewStyle().
		PaddingLeft(padH).
		PaddingRight(padH).
		PaddingTop(1)

	var body string
	switch m.screen {
	case ScreenForm:
		body = m.form.View()
	}

	content := contentStyle.Render(body)

	// Help bar pinned to the bottom.
	help := m.renderHelp(padH)
	contentLines := strings.Count(content, "\n") + 1
	helpLines := strings.Count(help, "\n") + 1
	gap := m.height - contentLines - helpLines
	if gap < 0 {
		gap = 0
	}

	return content + strings.Repeat("\n", gap) + help
}

func (m AppModel) renderHelp(padH int) string {
	keyBadge := lipgloss.NewStyle().
		Foreground(ColorFg).
		Background(ColorSubtle).
		Bold(true).
		Padding(0, 1)
	descStyle := lipgloss.NewStyle().
		Foreground(ColorMuted)

	pairs := []struct{ key, desc string }{
		{"tab", "navigate"},
		{"↑↓", "move"},
		{"ctrl+s", "submit"},
		{"ctrl+c", "quit"},
	}

	var parts []string
	for _, p := range pairs {
		parts = append(parts, keyBadge.Render(p.key)+" "+descStyle.Render(p.desc))
	}

	bar := strings.Join(parts, "   ")

	return lipgloss.NewStyle().
		PaddingLeft(padH).
		PaddingBottom(1).
		Render(bar)
}
