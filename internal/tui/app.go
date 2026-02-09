package tui

import (
	"fmt"
	"strings"

	"cheatr/internal/backend"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type interactiveModel struct {
	backend     backend.Backend
	input       textinput.Model
	search      searchModel
	results     []backend.SearchResult
	err         error
	selected    int
	width       int
	height      int
	isSearching bool
	lastQuery   string
	lastFilter  backend.SourceFilter
	styles      appStyles
}

type searchResultsMsg struct {
	query   string
	filter  backend.SourceFilter
	results []backend.SearchResult
	err     error
}

func RunInteractive(b backend.Backend) error {
	model := newInteractiveModel(b)
	if _, err := tea.NewProgram(model, tea.WithAltScreen()).Run(); err != nil {
		return err
	}
	return nil
}

func newInteractiveModel(b backend.Backend) interactiveModel {
	input := textinput.New()
	input.Prompt = "Search: "
	input.Placeholder = "Type to search..."
	input.Focus()
	input.CharLimit = 256
	input.Width = 48

	search := newSearchModel()
	return interactiveModel{
		backend:    b,
		input:      input,
		search:     search,
		lastFilter: search.filter,
		styles:     newAppStyles(),
	}
}

func (m interactiveModel) Init() tea.Cmd {
	return m.searchCmd()
}

func (m interactiveModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = typed.Width
		m.height = typed.Height
		m.input.Width = maxInt(10, typed.Width-12)
		return m, nil
	case searchResultsMsg:
		if typed.query != m.lastQuery || typed.filter != m.lastFilter {
			return m, nil
		}
		m.isSearching = false
		m.err = typed.err
		m.results = typed.results
		if len(m.results) == 0 {
			m.selected = 0
		} else if m.selected >= len(m.results) {
			m.selected = len(m.results) - 1
		}
		return m, nil
	case tea.KeyMsg:
		switch typed.String() {
		case "ctrl+c", "esc", "q":
			return m, tea.Quit
		case "tab":
			m.search.cycleSourceTabForward()
			m.lastFilter = m.search.filter
			return m, m.searchCmd()
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "j":
			if m.selected < len(m.results)-1 {
				m.selected++
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	query := strings.TrimSpace(m.input.Value())
	if query != m.lastQuery {
		m.lastQuery = query
		m.selected = 0
		return m, tea.Batch(cmd, m.searchCmd())
	}

	return m, cmd
}

func (m interactiveModel) searchCmd() tea.Cmd {
	query := strings.TrimSpace(m.input.Value())
	filter := m.search.filter
	m.isSearching = true
	m.lastQuery = query
	m.lastFilter = filter

	return func() tea.Msg {
		results, err := m.backend.Search(query, filter)
		return searchResultsMsg{query: query, filter: filter, results: results, err: err}
	}
}

func (m interactiveModel) View() string {
	parts := []string{
		m.styles.input.Render(m.input.View()),
		m.styles.tabs.Render(m.search.renderSourceTabs()),
	}

	if m.err != nil {
		parts = append(parts, m.styles.error.Render(fmt.Sprintf("Search failed: %v", m.err)))
		return strings.Join(parts, "\n")
	}

	rows := m.search.renderResults(m.results, m.selected, m.styles)
	if len(rows) == 0 {
		if m.isSearching {
			parts = append(parts, m.styles.dim.Render("Searching..."))
		} else {
			parts = append(parts, m.styles.dim.Render("No results."))
		}
		return strings.Join(parts, "\n")
	}

	parts = append(parts, strings.Join(rows, "\n"))
	return strings.Join(parts, "\n")
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
