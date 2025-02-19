package ui

import "github.com/charmbracelet/lipgloss"

func (m model) View() string {
	var content string

	if m.modal != nil {
		return m.modal.View()
	}

	switch m.activeTabIndex {
	case 0:
		content = m.installedTable.View()
	case 1:
		content = m.registryTable.View()
	}

	// Render the top bar with tabs and search
	tabsRow := RenderTabs(m, m.tabs, m.width)

	// Join the components vertically with the correct order
	return lipgloss.JoinVertical(
		lipgloss.Left,
		tabsRow,
		content,
	)
}
