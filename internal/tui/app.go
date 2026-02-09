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
	openRequest *viewerOpenRequest
}

type searchResultsMsg struct {
	query   string
	filter  backend.SourceFilter
	results []backend.SearchResult
	err     error
}

type openSelectionMsg struct {
	query    string
	source   string
	markdown string
	err      error
}

type viewerOpenRequest struct {
	query    string
	source   string
	markdown string
}

func RunInteractive(b backend.Backend) error {
	model := newInteractiveModel(b)
	finalModel, err := tea.NewProgram(model, tea.WithAltScreen()).Run()
	if err != nil {
		return err
	}

	if open := extractViewerOpenRequest(finalModel); open != nil {
		return RunPager(open.query, open.source, open.markdown)
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
		case "enter":
			if len(m.results) == 0 || m.selected < 0 || m.selected >= len(m.results) {
				return m, nil
			}
			return m, m.executeSelectionCmd(m.results[m.selected])
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
	case openSelectionMsg:
		if typed.err != nil {
			m.err = typed.err
			return m, nil
		}

		m.openRequest = &viewerOpenRequest{
			query:    typed.query,
			source:   typed.source,
			markdown: typed.markdown,
		}
		return m, tea.Quit
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

func extractViewerOpenRequest(model tea.Model) *viewerOpenRequest {
	switch typed := model.(type) {
	case interactiveModel:
		return typed.openRequest
	case *interactiveModel:
		return typed.openRequest
	default:
		return nil
	}
}

func (m interactiveModel) executeSelectionCmd(selected backend.SearchResult) tea.Cmd {
	return func() tea.Msg {
		return m.executeSelection(selected)
	}
}

func (m interactiveModel) executeSelection(selected backend.SearchResult) openSelectionMsg {
	switch selected.Kind {
	case backend.SearchEntry:
		return m.openEntrySelection(selected)
	case backend.SearchAction:
		return m.executeActionSelection(selected)
	default:
		return openSelectionMsg{err: fmt.Errorf("unsupported selection kind %q", selected.Kind)}
	}
}

func (m interactiveModel) openEntrySelection(selected backend.SearchResult) openSelectionMsg {
	if selected.Entry == nil {
		return openSelectionMsg{err: fmt.Errorf("selected row has no entry")}
	}

	if strings.TrimSpace(selected.Entry.Source) == backend.SourceDevDocs {
		slug := strings.TrimSpace(selected.Meta["slug"])
		if slug == "" {
			slug = strings.TrimSpace(selected.Entry.Topic)
		}

		entryPath := strings.TrimSpace(selected.Meta["path"])
		if entryPath == "" {
			entryPath = strings.TrimSpace(selected.Entry.Title)
		}

		if slug == "" || entryPath == "" {
			return openSelectionMsg{err: fmt.Errorf("devdocs selection missing slug/path metadata")}
		}

		markdown, err := m.resolveDocsSelection(slug, entryPath)
		if err != nil {
			return openSelectionMsg{err: err}
		}

		return openSelectionMsg{
			query:    strings.TrimSpace(selected.Entry.Title),
			source:   backend.SourceDevDocs,
			markdown: markdown,
		}
	}

	content := strings.TrimSpace(selected.Entry.Content)
	if content == "" {
		resolved, err := m.backend.GetEntry(&backend.Resolution{
			Source: selected.Entry.Source,
			Topic:  selected.Entry.Topic,
		})
		if err != nil {
			return openSelectionMsg{err: err}
		}
		content = strings.TrimSpace(resolved.Content)
	}

	if content == "" {
		return openSelectionMsg{err: backend.ErrResolutionNotFound}
	}

	return openSelectionMsg{
		query:    resultLabel(selected),
		source:   strings.TrimSpace(selected.Entry.Source),
		markdown: content,
	}
}

func (m interactiveModel) executeActionSelection(selected backend.SearchResult) openSelectionMsg {
	switch selected.Action {
	case backend.ActionBrowseDevDocs:
		return m.executeBrowseDevDocsAction(selected)
	case backend.ActionAskLLM:
		query := strings.TrimSpace(selected.Meta["query"])
		model := strings.TrimSpace(selected.Meta["model"])
		provider := strings.TrimSpace(selected.Meta["provider"])
		if model == "" {
			model = "configured model"
		}
		if provider == "" {
			provider = "configured provider"
		}

		return openSelectionMsg{
			query:  strings.TrimSpace(selected.Label),
			source: backend.SourceLLM,
			markdown: fmt.Sprintf(
				"# LLM action\n\n`Ask %s (%s)` is not fully wired in interactive mode yet.\n\nQuery: `%s`",
				model,
				provider,
				query,
			),
		}
	default:
		return openSelectionMsg{err: fmt.Errorf("unsupported action kind %q", selected.Action)}
	}
}

func (m interactiveModel) executeBrowseDevDocsAction(selected backend.SearchResult) openSelectionMsg {
	slug := strings.TrimSpace(selected.Meta["slug"])
	if slug != "" {
		resolved, err := m.backend.Resolve([]string{"docs", slug})
		if err != nil {
			return openSelectionMsg{err: err}
		}

		markdown := renderDevDocsCandidatesMarkdown(slug, resolved.Candidates)
		return openSelectionMsg{
			query:    strings.TrimSpace(selected.Label),
			source:   backend.SourceDevDocs,
			markdown: markdown,
		}
	}

	query := strings.TrimSpace(selected.Meta["query"])
	if query == "" {
		query = strings.TrimSpace(m.input.Value())
	}
	if query == "" {
		return openSelectionMsg{err: fmt.Errorf("browse action requires query or slug metadata")}
	}

	results, err := m.backend.Search(query, backend.FilterDevDocs)
	if err != nil {
		return openSelectionMsg{err: err}
	}

	markdown := renderDevDocsSearchMarkdown(query, results)
	return openSelectionMsg{
		query:    strings.TrimSpace(selected.Label),
		source:   backend.SourceDevDocs,
		markdown: markdown,
	}
}

func (m interactiveModel) resolveDocsSelection(slug, search string) (string, error) {
	resolved, err := m.backend.Resolve([]string{"docs", slug, search})
	if err != nil {
		return "", err
	}

	if strings.TrimSpace(resolved.Content) != "" {
		return resolved.Content, nil
	}

	if len(resolved.Candidates) > 0 {
		return renderDevDocsCandidatesMarkdown(slug, resolved.Candidates), nil
	}

	return "", backend.ErrResolutionNotFound
}

func renderDevDocsCandidatesMarkdown(slug string, candidates []backend.Candidate) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# %s DevDocs\n\n", strings.TrimSpace(slug)))
	if len(candidates) == 0 {
		builder.WriteString("No local entries found for this docset.")
		return builder.String()
	}

	builder.WriteString("Available entries:\n")
	for _, candidate := range candidates {
		title := strings.TrimSpace(candidate.Title)
		if title == "" {
			title = strings.TrimSpace(candidate.Path)
		}
		builder.WriteString(fmt.Sprintf("- %s (`%s`)\n", title, strings.TrimSpace(candidate.Path)))
	}

	return strings.TrimSpace(builder.String())
}

func renderDevDocsSearchMarkdown(query string, results []backend.SearchResult) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("# DevDocs search for %q\n\n", query))

	count := 0
	for _, result := range results {
		if result.Kind != backend.SearchEntry || result.Entry == nil {
			continue
		}

		slug := strings.TrimSpace(result.Meta["slug"])
		entryPath := strings.TrimSpace(result.Meta["path"])
		title := strings.TrimSpace(result.Entry.Title)
		if title == "" {
			title = entryPath
		}
		if slug == "" {
			slug = strings.TrimSpace(result.Entry.Topic)
		}

		builder.WriteString(fmt.Sprintf("- %s (`%s`, `%s`)\n", title, slug, entryPath))
		count++
	}

	if count == 0 {
		builder.WriteString("No local DevDocs entries matched.")
	}

	return strings.TrimSpace(builder.String())
}
