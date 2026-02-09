package tui

import "github.com/charmbracelet/lipgloss"

type appStyles struct {
	input        lipgloss.Style
	tabs         lipgloss.Style
	modeBar      lipgloss.Style
	group        lipgloss.Style
	row          lipgloss.Style
	selected     lipgloss.Style
	badge        lipgloss.Style
	dim          lipgloss.Style
	error        lipgloss.Style
	viewerHeader lipgloss.Style
	viewerBody   lipgloss.Style
	selectMark   lipgloss.Style
}

func newAppStyles() appStyles {
	return appStyles{
		input: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true),
		tabs: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("110")),
		modeBar: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("252")),
		group: lipgloss.NewStyle().
			Padding(1, 1, 0, 1).
			Bold(true).
			Foreground(lipgloss.Color("81")),
		row: lipgloss.NewStyle().
			Padding(0, 1),
		selected: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")),
		badge: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("62")),
		dim: lipgloss.NewStyle().
			Padding(1, 1, 0, 1).
			Foreground(lipgloss.Color("244")),
		error: lipgloss.NewStyle().
			Padding(1, 1, 0, 1).
			Foreground(lipgloss.Color("203")),
		viewerHeader: lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("24")),
		viewerBody: lipgloss.NewStyle().
			Padding(0, 1),
		selectMark: lipgloss.NewStyle().Bold(true),
	}
}
