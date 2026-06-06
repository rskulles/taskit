package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorPurple = lipgloss.Color("99")
	colorGray   = lipgloss.Color("240")
	colorGreen  = lipgloss.Color("42")
	colorRed    = lipgloss.Color("196")
	colorYellow = lipgloss.Color("220")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPurple).
			MarginBottom(1)

	styleBreadcrumb = lipgloss.NewStyle().
			Foreground(colorGray).
			MarginBottom(1)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorGray).
			MarginTop(1)

	styleSelected = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed)

	styleGray = lipgloss.NewStyle().Foreground(colorGray)

	styleStatus = map[string]lipgloss.Style{
		"new":         lipgloss.NewStyle().Foreground(colorGreen),
		"in_progress": lipgloss.NewStyle().Foreground(colorYellow),
		"done":        lipgloss.NewStyle().Foreground(colorGray),
		"blocked":     lipgloss.NewStyle().Foreground(colorRed),
		"archived":    lipgloss.NewStyle().Foreground(colorGray).Faint(true),
	}

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorGray).
			Padding(0, 1)
)

func statusBadge(s string) string {
	style, ok := styleStatus[s]
	if !ok {
		style = lipgloss.NewStyle()
	}
	return style.Render("[" + s + "]")
}
