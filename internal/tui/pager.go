package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type pagerModel struct {
	query         string
	source        string
	rawMarkdown   string
	rendered      string
	width         int
	height        int
	showHelp      bool
	renderErr     error
	headerStyle   lipgloss.Style
	queryStyle    lipgloss.Style
	badgeStyle    lipgloss.Style
	helpStyle     lipgloss.Style
	errorStyle    lipgloss.Style
	viewportStyle lipgloss.Style
	vp            viewport.Model
}

func newPagerModel(query, source, markdown string) pagerModel {
	vp := viewport.New(0, 0)
	vp.KeyMap = viewport.KeyMap{}
	vp.SetContent(strings.TrimSpace(markdown))

	return pagerModel{
		query:       strings.TrimSpace(query),
		source:      strings.TrimSpace(source),
		rawMarkdown: strings.TrimSpace(markdown),
		headerStyle: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")),
		queryStyle: lipgloss.NewStyle().Bold(true),
		badgeStyle: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("62")),
		helpStyle: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("248")),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("203")).
			Padding(1, 1),
		viewportStyle: lipgloss.NewStyle().Padding(0, 1),
		vp:            vp,
	}
}

func RunPager(query, source, markdown string) error {
	model := newPagerModel(query, source, markdown)
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		return err
	}
	return nil
}

func (m pagerModel) Init() tea.Cmd {
	return nil
}

func (m pagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.layoutViewport()
		m.renderMarkdownToViewport()
		return m, nil
	case tea.KeyMsg:
		switch typed.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			m.layoutViewport()
			m.renderMarkdownToViewport()
			return m, nil
		case "j", "down":
			m.vp.LineDown(1)
			return m, nil
		case "k", "up":
			m.vp.LineUp(1)
			return m, nil
		case "f", "pgdown", " ", "ctrl+f":
			m.vp.ViewDown()
			return m, nil
		case "b", "pgup", "ctrl+b":
			m.vp.ViewUp()
			return m, nil
		case "g", "home":
			m.vp.GotoTop()
			return m, nil
		case "G", "end":
			m.vp.GotoBottom()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m pagerModel) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	parts := []string{m.renderHeader()}
	if m.showHelp {
		parts = append(parts, m.helpStyle.Render("j/k scroll  f/b page  g/G top/bottom  q quit  ? help"))
	}

	if m.renderErr != nil {
		parts = append(parts, m.errorStyle.Render(fmt.Sprintf("render error: %v", m.renderErr)))
	} else {
		parts = append(parts, m.viewportStyle.Render(m.vp.View()))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (m *pagerModel) renderHeader() string {
	query := m.query
	if query == "" {
		query = "cheatr"
	}
	source := m.source
	if source == "" {
		source = "unknown"
	}

	left := m.queryStyle.Render(query)
	right := m.badgeStyle.Render(strings.ToUpper(source))
	available := m.width - lipgloss.Width(right) - 1
	if available < 0 {
		available = 0
	}
	left = lipgloss.NewStyle().Width(available).Render(left)

	return m.headerStyle.Width(m.width).Render(lipgloss.JoinHorizontal(lipgloss.Left, left, " ", right))
}

func (m *pagerModel) layoutViewport() {
	vertical := lipgloss.Height(m.renderHeader())
	if m.showHelp {
		vertical += lipgloss.Height(m.helpStyle.Render("x"))
	}

	contentHeight := m.height - vertical
	if contentHeight < 1 {
		contentHeight = 1
	}

	horizontalPadding := m.viewportStyle.GetHorizontalPadding()
	contentWidth := m.width - horizontalPadding
	if contentWidth < 1 {
		contentWidth = 1
	}

	m.vp.Width = contentWidth
	m.vp.Height = contentHeight
}

func (m *pagerModel) renderMarkdownToViewport() {
	if m.vp.Width < 1 {
		return
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.vp.Width),
	)
	if err != nil {
		m.renderErr = err
		m.vp.SetContent(m.rawMarkdown)
		return
	}

	rendered, err := renderer.Render(m.rawMarkdown)
	if err != nil {
		m.renderErr = err
		m.vp.SetContent(m.rawMarkdown)
		return
	}

	m.renderErr = nil
	m.rendered = rendered
	m.vp.SetContent(rendered)
}
