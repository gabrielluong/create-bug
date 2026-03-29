package form

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gabrielluong/create-bug/internal/bugzilla"
	"github.com/gabrielluong/create-bug/internal/client"
	"github.com/gabrielluong/create-bug/internal/config"
	"github.com/gabrielluong/create-bug/internal/history"
	"github.com/gabrielluong/create-bug/internal/tui/component"
)

// Focus zone identifiers.
type focusZone int

const (
	zoneSummary focusZone = iota
	zoneComponent
	zoneDescription
	zoneBlocks
	zoneDependsOn
	zoneSubmit
	zoneCount // sentinel
)

// formState tracks the current lifecycle of the form.
type formState int

const (
	stateEditing formState = iota
	stateSubmitting
	stateSuccess
	stateError
)

// submitResultMsg is sent when the bug submission completes.
type submitResultMsg struct {
	result *client.CreateBugResult
	err    error
}

// Model is the Bubbletea model for the bug creation form.
type Model struct {
	cfg   *config.Config
	focus focusZone
	state formState

	summary     textinput.Model
	component   component.FuzzySelect
	description textarea.Model
	blocks      component.BugIDInput
	dependsOn   component.BugIDInput
	spinner     spinner.Model

	result *client.CreateBugResult
	err    error

	width  int
	height int
}

// New creates a new form Model.
func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Bug summary..."
	ti.CharLimit = 256
	ti.Focus()

	// Build component selector from known product components.
	product := cfg.Defaults.Product
	items := bugzilla.ProductComponents[product]
	cs := component.NewFuzzySelect("Select component...", items)
	if cfg.Defaults.Component != "" {
		cs.SetValue(cfg.Defaults.Component)
	}

	ta := textarea.New()
	ta.Placeholder = "Steps to reproduce, expected results, actual results..."
	ta.SetHeight(6)
	ta.ShowLineNumbers = false

	// Load history for bug ID autocomplete.
	entries, _ := history.Load()
	bl := component.NewBugIDInput("Bug IDs this blocks (e.g. 123, 456)...", entries)
	dep := component.NewBugIDInput("Bug IDs this depends on (e.g. 123, 456)...", entries)

	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return Model{
		cfg:         cfg,
		focus:       zoneSummary,
		state:       stateEditing,
		summary:     ti,
		component:   cs,
		description: ta,
		blocks:      bl,
		dependsOn:   dep,
		spinner:     sp,
	}
}

// SetSize updates the form dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	contentWidth := width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}
	m.summary.Width = contentWidth
	m.description.SetWidth(contentWidth)
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case submitResultMsg:
		if msg.err != nil {
			m.state = stateError
			m.err = msg.err
		} else {
			m.state = stateSuccess
			m.result = msg.result
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m.updateChildren(msg)
}

func (m Model) handleKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	// Post-submit key handling.
	if m.state == stateSuccess {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+n"))):
			m.resetForNextBug()
			return m, textinput.Blink
		case key.Matches(msg, key.NewBinding(key.WithKeys("q"))):
			return m, tea.Quit
		}
		return m, nil
	}

	// Error state: any key returns to editing.
	if m.state == stateError {
		m.state = stateEditing
		m.err = nil
		return m, nil
	}

	// Don't handle keys while submitting.
	if m.state == stateSubmitting {
		return m.updateChildren(msg)
	}

	// Tab navigation.
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab"))):
		m.nextFocus()
		return m, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("shift+tab"))):
		m.prevFocus()
		return m, nil
	}

	// Up/Down arrow navigation between fields (when no dropdown is open
	// and the field doesn't need vertical movement internally).
	if msg.Type == tea.KeyUp && !m.fieldCapturesVertical() {
		m.prevFocus()
		return m, nil
	}
	if msg.Type == tea.KeyDown && !m.fieldCapturesVertical() {
		m.nextFocus()
		return m, nil
	}

	// Submit shortcut.
	if key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+s"))) {
		return m.submit()
	}

	// Enter on submit button.
	if m.focus == zoneSubmit && msg.Type == tea.KeyEnter {
		return m.submit()
	}

	return m.updateChildren(msg)
}

func (m Model) updateChildren(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.focus == zoneSummary {
		var cmd tea.Cmd
		m.summary, cmd = m.summary.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.focus == zoneComponent {
		var cmd tea.Cmd
		m.component, cmd = m.component.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.focus == zoneDescription {
		var cmd tea.Cmd
		m.description, cmd = m.description.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.focus == zoneBlocks {
		var cmd tea.Cmd
		m.blocks, cmd = m.blocks.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.focus == zoneDependsOn {
		var cmd tea.Cmd
		m.dependsOn, cmd = m.dependsOn.Update(msg)
		cmds = append(cmds, cmd)
	}
	if m.state == stateSubmitting {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// fieldCapturesVertical reports whether the focused field needs Up/Down
// for its own navigation (open dropdown or multiline textarea).
func (m *Model) fieldCapturesVertical() bool {
	switch m.focus {
	case zoneComponent:
		return m.component.IsOpen()
	case zoneDescription:
		return true // textarea always uses Up/Down for line movement
	case zoneBlocks:
		return m.blocks.IsOpen()
	case zoneDependsOn:
		return m.dependsOn.IsOpen()
	}
	return false
}

func (m *Model) nextFocus() {
	m.setFocus((m.focus + 1) % zoneCount)
}

func (m *Model) prevFocus() {
	m.setFocus((m.focus - 1 + zoneCount) % zoneCount)
}

func (m *Model) setFocus(zone focusZone) {
	m.focus = zone
	if zone == zoneSummary {
		m.summary.Focus()
	} else {
		m.summary.Blur()
	}
	if zone == zoneComponent {
		m.component.Focus()
	} else {
		m.component.Blur()
	}
	if zone == zoneDescription {
		m.description.Focus()
	} else {
		m.description.Blur()
	}
	if zone == zoneBlocks {
		m.blocks.Focus()
	} else {
		m.blocks.Blur()
	}
	if zone == zoneDependsOn {
		m.dependsOn.Focus()
	} else {
		m.dependsOn.Blur()
	}
}

func (m Model) submit() (Model, tea.Cmd) {
	summary := strings.TrimSpace(m.summary.Value())
	if summary == "" {
		m.state = stateError
		m.err = fmt.Errorf("summary is required")
		return m, nil
	}

	if m.cfg.APIKey == "" {
		m.state = stateError
		m.err = fmt.Errorf("BUGZILLA_API_KEY is not set")
		return m, nil
	}

	if m.cfg.BaseURL == "" {
		m.state = stateError
		m.err = fmt.Errorf("BUGZILLA_URL is not set")
		return m, nil
	}

	defaults := m.cfg.Defaults
	product := defaults.Product
	comp := m.component.Value()
	version := defaults.Version
	bugType := defaults.Type
	priority := defaults.Priority
	severity := defaults.Severity
	platform := defaults.Platform
	opSys := defaults.OS

	if product == "" {
		m.state = stateError
		m.err = fmt.Errorf("product is required (set in config defaults)")
		return m, nil
	}
	if comp == "" {
		m.state = stateError
		m.err = fmt.Errorf("component is required")
		return m, nil
	}
	if version == "" {
		version = "unspecified"
	}
	if bugType == "" {
		bugType = "defect"
	}

	// Resolve component via fuzzy matching.
	resolved, err := bugzilla.ResolveComponent(product, comp)
	if err != nil {
		m.state = stateError
		m.err = err
		return m, nil
	}

	m.state = stateSubmitting

	params := client.CreateBugParams{
		Product:     product,
		Component:   resolved,
		Summary:     summary,
		Version:     version,
		Type:        bugType,
		Description: strings.TrimSpace(m.description.Value()),
		Priority:    priority,
		Severity:    severity,
		Platform:    platform,
		OpSys:       opSys,
		Blocks:      m.blocks.IDs(),
		DependsOn:   m.dependsOn.IDs(),
	}

	cfg := m.cfg
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			result, err := bugzilla.CreateBug(cfg, params)
			if err != nil {
				return submitResultMsg{err: err}
			}

			// Save to history (fire-and-forget).
			url := fmt.Sprintf("%s/show_bug.cgi?id=%d", strings.TrimRight(cfg.BaseURL, "/"), result.ID)
			_ = history.Append(history.Entry{
				ID:        result.ID,
				Summary:   params.Summary,
				Product:   params.Product,
				Component: params.Component,
				URL:       url,
				CreatedAt: time.Now(),
			}, cfg.HistorySize)

			return submitResultMsg{result: result}
		},
	)
}

func (m *Model) resetForNextBug() {
	m.summary.SetValue("")
	m.description.SetValue("")
	// Component is intentionally preserved for multi-bug filing.
	// Reload history so newly filed bugs appear in autocomplete.
	entries, _ := history.Load()
	m.blocks = component.NewBugIDInput("Bug IDs this blocks (e.g. 123, 456)...", entries)
	m.dependsOn = component.NewBugIDInput("Bug IDs this depends on (e.g. 123, 456)...", entries)
	m.state = stateEditing
	m.result = nil
	m.err = nil
	m.setFocus(zoneSummary)
}

// View renders the form.
func (m Model) View() string {
	var b strings.Builder

	switch m.state {
	case stateSubmitting:
		b.WriteString("\n\n")
		b.WriteString("  " + m.spinner.View() + spinnerTextStyle.Render(" Filing bug...") + "\n")
		return b.String()

	case stateSuccess:
		url := fmt.Sprintf("%s/show_bug.cgi?id=%d", strings.TrimRight(m.cfg.BaseURL, "/"), m.result.ID)
		b.WriteString("\n")
		badge := successBadge.Render(" BUG #" + fmt.Sprintf("%d", m.result.ID) + " ")
		b.WriteString("  " + badge + "  " + successText.Render("created") + "\n\n")
		b.WriteString("  " + urlStyle.Render(url) + "\n\n")

		divider := dividerStyle.Render(strings.Repeat("─", 40))
		b.WriteString("  " + divider + "\n\n")
		b.WriteString("  " + helpKeyBadge.Render("ctrl+n") + " " + mutedStyle.Render("file another") + "    " + helpKeyBadge.Render("q") + " " + mutedStyle.Render("quit") + "\n")
		return b.String()

	case stateError:
		b.WriteString("\n\n")
		b.WriteString("  " + errorIcon.Render(" ! ") + "  " + errorText.Render(m.err.Error()) + "\n\n")
		b.WriteString("  " + mutedStyle.Render("Press any key to continue") + "\n")
		return b.String()
	}

	// Primary fields.
	b.WriteString(m.renderField("Summary", m.summary.View(), m.focus == zoneSummary))
	b.WriteString(m.renderField("Component", m.component.View(), m.focus == zoneComponent))
	b.WriteString(m.renderField("Description", m.description.View(), m.focus == zoneDescription))
	b.WriteString(m.renderField("Blocks", m.blocks.View(), m.focus == zoneBlocks))
	b.WriteString(m.renderField("Depends on", m.dependsOn.View(), m.focus == zoneDependsOn))

	// Secondary field defaults as pill badges.
	defaults := m.cfg.Defaults
	var pills []string
	if defaults.Product != "" {
		pills = append(pills, renderPill("Product", defaults.Product))
	}
	if defaults.Type != "" {
		pills = append(pills, renderPill("Type", defaults.Type))
	}
	if defaults.Priority != "" {
		pills = append(pills, renderPill("Priority", defaults.Priority))
	}
	if defaults.Severity != "" {
		pills = append(pills, renderPill("Severity", defaults.Severity))
	}
	if len(pills) > 0 {
		b.WriteString("  " + strings.Join(pills, "  ") + "\n\n")
	}

	// Submit button.
	if m.focus == zoneSubmit {
		b.WriteString("  " + buttonActiveStyle.Render("  Submit  "))
	} else {
		b.WriteString("  " + buttonStyle.Render("  Submit  "))
	}
	b.WriteString("\n")

	return b.String()
}

func (m Model) renderField(label, content string, active bool) string {
	var b strings.Builder

	if active {
		b.WriteString(focusBar.Render("│") + " ")
		b.WriteString(activeLabelStyle.Render(label))
	} else {
		b.WriteString("  ")
		b.WriteString(labelStyle.Render(label))
	}
	b.WriteString("\n")

	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if active {
			b.WriteString(focusBar.Render("│") + " " + line)
		} else {
			b.WriteString("  " + line)
		}
		if i < len(lines)-1 {
			b.WriteString("\n")
		}
	}
	b.WriteString("\n\n")

	return b.String()
}

func renderPill(label, value string) string {
	return pillLabelStyle.Render(" "+label+" ") + pillValueStyle.Render(" "+value+" ")
}

// Local styles.
var (
	focusBar = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8B5CF6")).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B"))

	activeLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#C4B5FD"))

	successBadge = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#022C22")).
			Background(lipgloss.Color("#34D399"))

	successText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#34D399"))

	urlStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Underline(true)

	errorIcon = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FEF2F2")).
			Background(lipgloss.Color("#F87171"))

	errorText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F87171"))

	spinnerTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#94A3B8"))

	mutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#64748B"))

	helpKeyBadge = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0")).
			Background(lipgloss.Color("#334155")).
			Bold(true).
			Padding(0, 1)

	dividerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#334155"))

	pillLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Background(lipgloss.Color("#1E293B"))

	pillValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#E2E8F0")).
			Background(lipgloss.Color("#334155"))

	buttonActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#7C3AED"))

	buttonStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#94A3B8")).
			Background(lipgloss.Color("#1E293B"))
)
