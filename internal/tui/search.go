package tui

import (
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
