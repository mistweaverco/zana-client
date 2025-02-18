package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
)

var (
	// General styles
	normal    = lipgloss.Color("#EEEEEE")
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	base = lipgloss.NewStyle().Foreground(normal)

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	packagedInstalledStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
	updateAvailableStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
	missingInRegistryStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	checkingForUpdatesStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#73F59F"))
	installedVersionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

type TabType int

const (
	TabNormal TabType = iota
	TabSearch
)

// Tab struct for modular tabs
type Tab struct {
	Title    string
	IsActive bool
	Id       string
	Type     TabType
}

func (t Tab) Render() string {
	style := lipgloss.NewStyle().
		Align(lipgloss.Center).
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "┴",
			BottomRight: "┴",
		}, true).
		BorderForeground(highlight).
		Padding(0, 1)

	if t.IsActive {
		style = style.Border(lipgloss.Border{
			Top:         "─",
			Bottom:      " ",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "┘",
			BottomRight: "└",
		}, true)
	}

	return style.Render(t.Title)
}

// RenderTabs creates the tab row with a full-width bottom line
func RenderTabs(m model, tabs []Tab, totalWidth int) string {
	var renderedTabs []string
	for _, tab := range tabs {
		renderedTabs = append(renderedTabs, tab.Render())
	}

	row := lipgloss.JoinHorizontal(lipgloss.Bottom, renderedTabs...)

	// Style for the search input
	searchStyle := lipgloss.NewStyle().
		Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "┴",
			BottomRight: "┴",
		}, true).
		BorderForeground(highlight).
		Padding(0, 1)

	searchView := searchStyle.Render(m.searchInput.View())

	// Calculate space between tabs and search
	gapWidth := totalWidth - lipgloss.Width(row) - lipgloss.Width(searchView)
	if gapWidth > 0 {
		gap := strings.Repeat("─", gapWidth)
		row = lipgloss.JoinHorizontal(
			lipgloss.Bottom,
			row,
			lipgloss.NewStyle().Foreground(highlight).Render(gap),
			searchView,
		)
	}

	return row
}

// Item struct for the list
type localPackageItem struct {
	title, desc, sourceId, version string
	updateAvailable                bool
}

func (i localPackageItem) Title() string       { return i.title }
func (i localPackageItem) Description() string { return i.desc }
func (i localPackageItem) FilterValue() string { return i.title }

// Item struct for the list
type registryPackageItem struct {
	title, desc, sourceId, version string
	installed                      bool
}

func (i registryPackageItem) Title() string       { return i.title }
func (i registryPackageItem) Description() string { return i.desc }
func (i registryPackageItem) FilterValue() string { return i.title }

// Main model
type model struct {
	installedTable table.Model
	registryTable  table.Model
	tabs           []Tab
	activeTabIndex int
	searchInput    textinput.Model

	width, height  int
	spinner        spinner.Model
	spinnerVisible bool
	spinnerMessage string
	updating       bool
	updated        bool
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Limit maximum window size to 80x25
		width := min(msg.Width, 80)
		height := min(msg.Height, 25)

		m.width = width
		m.height = height

		// Calculate dynamic column widths
		nameWidth := int(float64(width) * 0.6) // 60% of width for name
		versionWidth := width - nameWidth - 2  // Remaining width for version, account for borders

		// Update column widths
		m.installedTable.SetColumns([]table.Column{
			{Title: "Name", Width: nameWidth},
			{Title: "Version", Width: versionWidth},
		})
		m.registryTable.SetColumns([]table.Column{
			{Title: "Name", Width: nameWidth},
			{Title: "Version", Width: versionWidth},
		})

		// Update table dimensions
		m.installedTable.SetWidth(width)
		m.registryTable.SetWidth(width)

		// Set height to leave room for tabs and borders
		tableHeight := height - 4 // Account for tabs and borders
		m.installedTable.SetHeight(tableHeight)
		m.registryTable.SetHeight(tableHeight)

		return m, nil

	case tea.KeyMsg:
		if !m.searchInput.Focused() {
			switch msg.String() {
			case "tab", "shift+tab":
				// Handle tab switching
				m.activeTabIndex = (m.activeTabIndex + 1) % len(m.tabs)
				// Update active state of tabs
				for i := range m.tabs {
					m.tabs[i].IsActive = (i == m.activeTabIndex)
				}
				return m, nil
			case "/":
				m.searchInput.Focus()
				return m, m.searchInput.Focus()
			case "esc":
				m.searchInput.Blur()
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}

			// Handle table navigation when search is not focused
			switch m.activeTabIndex {
			case 0:
				m.installedTable, cmd = m.installedTable.Update(msg)
			case 1:
				m.registryTable, cmd = m.registryTable.Update(msg)
			}
		} else {
			switch msg.String() {
			case "esc":
				m.searchInput.Blur()
				return m, nil
			}
			// Handle search input updates
			m.searchInput, cmd = m.searchInput.Update(msg)

			// Filter table rows based on search
			if m.activeTabIndex == 0 {
				m.filterInstalledTable(m.searchInput.Value())
			} else {
				m.filterRegistryTable(m.searchInput.Value())
			}
		}
	}

	return m, cmd
}

// Helper function to filter installed table
func (m *model) filterInstalledTable(query string) {
	if query == "" {
		m.updateInstalledTableRows(m.getInstalledPackages()) // Show all rows
		return
	}

	filtered := []localPackageItem{}
	for _, item := range m.getInstalledPackages() {
		if strings.Contains(strings.ToLower(item.title), strings.ToLower(query)) {
			filtered = append(filtered, item)
		}
	}
	m.updateInstalledTableRows(filtered)
}

// Helper function to filter registry table
func (m *model) filterRegistryTable(query string) {
	if query == "" {
		m.updateRegistryTableRows(m.getRegistryPackages()) // Show all rows
		return
	}

	filtered := []registryPackageItem{}
	for _, item := range m.getRegistryPackages() {
		if strings.Contains(strings.ToLower(item.title), strings.ToLower(query)) {
			filtered = append(filtered, item)
		}
	}
	m.updateRegistryTableRows(filtered)
}

func (m *model) getInstalledPackages() []localPackageItem {
	return []localPackageItem{}
}

func (m *model) getRegistryPackages() []registryPackageItem {
	data := registry_parser.GetData(false)
	regItems := []registryPackageItem{}

	for _, item := range data {
		regItems = append(regItems, registryPackageItem{
			title:     item.Name,
			desc:      item.Description,
			sourceId:  item.Source.ID,
			version:   item.Version,
			installed: false,
		})
	}

	return regItems
}

func (m model) View() string {
	var content string

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

func (m model) setupRegistryList() table.Model {
	registryPackages := registry_parser.GetData(false)
	registryItems := []registryPackageItem{}

	for _, registryPackage := range registryPackages {
		isInstalled := local_packages_parser.IsPackageInstalled(registryPackage.Source.ID)
		title := registryPackage.Name
		if isInstalled {
			title = title + " " + packagedInstalledStyle.Render("Installed")
		}
		regItem := registryPackageItem{
			sourceId:  registryPackage.Source.ID,
			title:     title,
			desc:      registryPackage.Description,
			version:   registryPackage.Version,
			installed: isInstalled,
		}
		registryItems = append(registryItems, regItem)
	}

	return m.registryTable
}

func (m model) initLists() (tea.Model, tea.Cmd) {
	m.updating = true
	localPackages := local_packages_parser.GetData(false)
	installedItems := []localPackageItem{}

	for _, localPackage := range localPackages.Packages {
		regItem := registry_parser.GetBySourceId(localPackage.SourceID)
		updateAvailable, remoteVersion := updater.CheckIfUpdateIsAvailable(localPackage.Version, regItem.Source.ID)

		localItem := localPackageItem{
			sourceId:        localPackage.SourceID,
			version:         localPackage.Version,
			updateAvailable: updateAvailable,
		}

		if regItem.Source.ID == "" {
			localItem.title = localPackage.SourceID + " " + installedVersionStyle.Render(localPackage.Version)
			localItem.desc = missingInRegistryStyle.Render("Not found in registry")
		} else if updateAvailable {
			localItem.title = regItem.Name + " " + installedVersionStyle.Render(localPackage.Version) + " " + updateAvailableStyle.Render("Update available: "+remoteVersion)
			localItem.desc = regItem.Description
		} else {
			localItem.title = regItem.Name + " " + installedVersionStyle.Render(localPackage.Version)
			localItem.desc = regItem.Description
		}

		installedItems = append(installedItems, localItem)
	}

	m.registryTable = m.setupRegistryList()

	m.spinnerMessage = "Updates checked"
	m.spinnerVisible = false
	m.updated = true

	return m, nil
}

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
		table.WithHeight(20), // Reduced height to fit in 25 lines
	)

	registryTable := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20), // Reduced height to fit in 25 lines
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
	}

	// Initialize the tables with data
	m.installedTable.SetRows([]table.Row{})
	regItems := getRegistryItemsData()
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

func Show() {
	m := initialModel()

	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

// Helper function to update table rows from package items
func (m *model) updateInstalledTableRows(items []localPackageItem) {
	rows := make([]table.Row, len(items))
	for i, item := range items {
		rows[i] = table.Row{
			item.title,
			item.version,
			item.desc,
		}
	}
	m.installedTable.SetRows(rows)
}

func (m *model) updateRegistryTableRows(items []registryPackageItem) {
	rows := make([]table.Row, len(items))
	for i, item := range items {
		rows[i] = table.Row{
			item.title,
			item.version,
		}
	}
	m.registryTable.SetRows(rows)
}

func getRegistryItemsData() []registryPackageItem {
	registryItems := []registryPackageItem{}

	for _, item := range registry_parser.GetData(true) {
		registryItems = append(registryItems, registryPackageItem{
			title:     item.Name,
			desc:      item.Description,
			sourceId:  item.Source.ID,
			version:   item.Version,
			installed: false,
		})
	}
	return registryItems
}

// Message types
type registryMsg struct{}
type localPackagesMsg struct{}

func (m model) handleRegistryMsg(msg registryMsg) (tea.Model, tea.Cmd) {
	m.spinnerVisible = false
	m.updateRegistryTableRows(getRegistryItemsData())
	return m, nil
}

func (m model) handleLocalPackagesMsg(msg localPackagesMsg) (tea.Model, tea.Cmd) {
	localItems := []localPackageItem{}

	for _, item := range local_packages_parser.GetData(true).Packages {
		registryItem := registry_parser.GetBySourceId(item.SourceID)
		updateAvailable, _ := updater.CheckIfUpdateIsAvailable(item.Version, item.SourceID)

		localItems = append(localItems, localPackageItem{
			title:           registryItem.Name,
			desc:            registryItem.Description,
			sourceId:        item.SourceID,
			version:         item.Version,
			updateAvailable: updateAvailable,
		})
	}

	// Update the installed table with the new data
	m.updateInstalledTableRows(localItems)
	return m, nil
}

func (m model) handleSpinnerTick() (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.spinnerVisible {
		m.spinner, cmd = m.spinner.Update(spinner.TickMsg{})
	}
	return m, cmd
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
