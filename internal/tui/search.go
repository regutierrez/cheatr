package tui

import (
	"fmt"
	"strings"

	"cheatr/internal/backend"
	tea "github.com/charmbracelet/bubbletea"
)

type sourceTab struct {
	label  string
	filter backend.SourceFilter
}

var sourceTabs = []sourceTab{
	{label: "All", filter: backend.FilterNone},
	{label: "LXIYM", filter: backend.FilterLXIYM},
	{label: "Devhints", filter: backend.FilterDevhints},
	{label: "tldr", filter: backend.FilterTldr},
	{label: "DevDocs", filter: backend.FilterDevDocs},
}

type searchModel struct {
	activeTab int
	filter    backend.SourceFilter
}

func newSearchModel() searchModel {
	return searchModel{activeTab: 0, filter: sourceTabs[0].filter}
}

func (m *searchModel) handleSearchKey(msg tea.KeyMsg) bool {
	if msg.Type != tea.KeyTab {
		return false
	}
	m.cycleSourceTabForward()
	return true
}

func (m *searchModel) cycleSourceTabForward() {
	m.activeTab = (m.activeTab + 1) % len(sourceTabs)
	m.filter = sourceTabs[m.activeTab].filter
}

func (m *searchModel) cycleSourceTabBackward() {
	m.activeTab--
	if m.activeTab < 0 {
		m.activeTab = len(sourceTabs) - 1
	}
	m.filter = sourceTabs[m.activeTab].filter
}

func (m searchModel) renderSourceTabs() string {
	labels := make([]string, 0, len(sourceTabs))
	for i, tab := range sourceTabs {
		if i == m.activeTab {
			labels = append(labels, "[ "+tab.label+" ]")
			continue
		}
		labels = append(labels, tab.label)
	}
	return strings.Join(labels, " ")
}

func (m searchModel) renderResults(results []backend.SearchResult, selected int, styles appStyles) []string {
	if len(results) == 0 {
		return nil
	}

	lines := make([]string, 0, len(results)+4)
	showGroups := m.filter == backend.FilterNone

	if !showGroups {
		for i, result := range results {
			lines = append(lines, renderResultLine(result, i == selected, styles))
		}
		return lines
	}

	grouped := groupResultsBySource(results)
	order := []string{
		backend.SourceLXIYM,
		backend.SourceDevhints,
		backend.SourceTldr,
		backend.SourceDevDocs,
		backend.SourceLLM,
	}

	globalIndex := 0
	for _, source := range order {
		bucket := grouped[source]
		if len(bucket) == 0 {
			continue
		}

		lines = append(lines, styles.group.Render(fmt.Sprintf("%s (%d)", sourceLabel(source), len(bucket))))
		for _, result := range bucket {
			lines = append(lines, renderResultLine(result, globalIndex == selected, styles))
			globalIndex++
		}
	}

	return lines
}

func groupResultsBySource(results []backend.SearchResult) map[string][]backend.SearchResult {
	grouped := make(map[string][]backend.SearchResult, len(sourceTabs))
	for _, result := range results {
		source := strings.TrimSpace(result.Source)
		if source == "" {
			source = "unknown"
		}
		grouped[source] = append(grouped[source], result)
	}
	return grouped
}

func renderResultLine(result backend.SearchResult, selected bool, styles appStyles) string {
	text := resultLabel(result)
	source := strings.TrimSpace(result.Source)
	if source == "" {
		source = "unknown"
	}

	line := fmt.Sprintf("  %s %s", text, styles.badge.Render(source))
	if selected {
		line = "> " + strings.TrimLeft(line, " ")
		return styles.selected.Render(line)
	}

	return styles.row.Render(line)
}

func resultLabel(result backend.SearchResult) string {
	if result.Kind == backend.SearchAction {
		return strings.TrimSpace(result.Label)
	}

	if result.Entry == nil {
		return "(missing entry)"
	}

	topic := strings.TrimSpace(result.Entry.Topic)
	title := strings.TrimSpace(result.Entry.Title)
	if topic == "" {
		return title
	}
	if title == "" || strings.EqualFold(topic, title) {
		return topic
	}
	return fmt.Sprintf("%s - %s", topic, title)
}

func sourceLabel(source string) string {
	switch source {
	case backend.SourceLXIYM:
		return "LXIYM"
	case backend.SourceDevhints:
		return "Devhints"
	case backend.SourceTldr:
		return "tldr"
	case backend.SourceDevDocs:
		return "DevDocs"
	case backend.SourceLLM:
		return "LLM"
	default:
		if strings.TrimSpace(source) == "" {
			return "Unknown"
		}
		return source
	}
}
