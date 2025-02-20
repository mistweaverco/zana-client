package ui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

func initialModel() model {
	// Define table columns with proportional widths
	columns := []table.Column{
		{Title: "Name", Width: 0},    // Width will be set dynamically
		{Title: "Version", Width: 0}, // Width will be set dynamically
	}

	// Initialize tables with default styles
	installedTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	registryTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
	)

	// Style the tables
	baseStyle := table.DefaultStyles()
	baseStyle.Header = baseStyle.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	baseStyle.Selected = baseStyle.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)

	installedTable.SetStyles(baseStyle)
	registryTable.SetStyles(baseStyle)

	// Initialize search input
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Width = 20
	ti.PromptStyle = lipgloss.NewStyle().Foreground(highlight)
	ti.Prompt = "🔍 "

	m := model{
		installedTable: installedTable,
		registryTable:  registryTable,
		tabs: []Tab{
			{Title: "Installed", IsActive: true, Id: "installed"},
			{Title: "Registry", Id: "registry"},
		},
		spinner:        spinner.New(spinner.WithSpinner(spinner.Points)),
		spinnerVisible: true,
		spinnerMessage: "Checking for updates",
		searchInput:    ti,
		currentView:    "installed",
	}

	installedItems := getLocalPackagesData()
	m.visibleInstalledData = installedItems
	installedRows := make([]table.Row, 0, len(installedItems))
	for _, item := range installedItems {
		version := item.version
		if item.updateAvailable {
			version = " " + version + " -> " + item.remoteVersion
		}
		installedRows = append(installedRows, table.Row{
			item.title,
			version,
		})
	}
	m.installedTable.SetRows(installedRows)
	regItems := getRegistryItemsData()
	m.visibleRegistryData = regItems
	registryRows := make([]table.Row, 0, len(regItems))
	for _, item := range regItems {
		registryRows = append(registryRows, table.Row{
			item.title,
			item.version,
		})
	}
	m.registryTable.SetRows(registryRows)

	return m
}
