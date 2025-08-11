package ui

import (
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
	"github.com/mistweaverco/zana-client/internal/modal"
)

var (
	// General styles
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
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
	title, desc, sourceId, version, remoteVersion string
	updateAvailable                               bool
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

	visibleInstalledData []localPackageItem
	visibleRegistryData  []registryPackageItem

	width, height  int
	spinner        spinner.Model
	spinnerVisible bool
	spinnerMessage string
	currentView    string
	modal          *modal.Modal
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) showModal(msg string, t string) (tea.Model, tea.Cmd) {
	newModal := modal.New(msg, t)
	m.modal = &newModal
	*m.modal, _ = m.modal.Update(tea.WindowSizeMsg{Width: m.width, Height: m.height})
	return m, nil
}

// Helper function to filter installed table
func (m *model) filterInstalledTable(query string) {
	if query == "" {
		m.updateInstalledTableRows(getLocalPackagesData()) // Show all rows
		return
	}

	filtered := []localPackageItem{}
	for _, item := range getLocalPackagesData() {
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

func (m *model) getRegistryPackages() []registryPackageItem {
	data := registry_parser.GetData(false)
	regItems := []registryPackageItem{}

	for _, item := range data {
		regItems = append(regItems, registryPackageItem{
			title:     item.Name,
			desc:      item.Description,
			sourceId:  item.Source.ID,
			version:   strings.TrimSpace(item.Version),
			installed: false,
		})
	}

	return regItems
}

// Helper function to update table rows from package items
func (m *model) updateInstalledTableRows(items []localPackageItem) {
	rows := make([]table.Row, len(items))
	m.visibleInstalledData = items

	for i, item := range items {
		// Get the version column width and truncate if necessary
		versionWidth := m.installedTable.Columns()[1].Width
		version := item.version
		if item.updateAvailable {
			version = "🔼 " + version
		}
		truncatedVersion := truncateString(version, versionWidth)

		rows[i] = table.Row{
			item.title,
			truncatedVersion,
		}
	}

	m.installedTable.SetRows(rows)
}

func (m *model) updateRegistryTableRows(items []registryPackageItem) {
	rows := make([]table.Row, len(items))
	m.visibleRegistryData = items
	for i, item := range items {
		// Get the version column width and truncate if necessary
		versionWidth := m.registryTable.Columns()[1].Width
		truncatedVersion := truncateString(item.version, versionWidth)

		rows[i] = table.Row{
			item.title,
			truncatedVersion,
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
			version:   strings.TrimSpace(item.Version),
			installed: false,
		})
	}
	return registryItems
}

func getLocalPackagesData() []localPackageItem {
	localItems := []localPackageItem{}
	localPackages := local_packages_parser.GetData(true).Packages
	registryPackages := registry_parser.GetData(true)

	// Create a map for quick lookup of local package versions by sourceID
	localPackageMap := make(map[string]string)
	for _, localPkg := range localPackages {
		localPackageMap[localPkg.SourceID] = localPkg.Version
	}

	// Iterate over registry entries
	for _, registryItem := range registryPackages {
		// Check if this registry item is installed locally
		if localVersion, isInstalled := localPackageMap[registryItem.Source.ID]; isInstalled {
			updateAvailable, _ := updater.CheckIfUpdateIsAvailable(localVersion, registryItem.Version)
			localItems = append(localItems, localPackageItem{
				title:           registryItem.Name,
				desc:            registryItem.Description,
				sourceId:        registryItem.Source.ID,
				version:         localVersion,
				remoteVersion:   registryItem.Version,
				updateAvailable: updateAvailable,
			})
		}
	}

	// if an update is available sort the list by update available first
	if len(localItems) > 0 {
		sort.Slice(localItems, func(i, j int) bool {
			// Primary sort: updateAvailable
			if localItems[i].updateAvailable && !localItems[j].updateAvailable {
				return true
			}
			if !localItems[i].updateAvailable && localItems[j].updateAvailable {
				return false
			}

			// Secondary sort: Title (if updateAvailable is the same)
			return localItems[i].title < localItems[j].title
		})
	}

	return localItems
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	// If string is already shorter than max length, return as is
	if len(s) <= maxLen {
		return s
	}

	// Ensure we have room for ellipsis
	if maxLen <= 3 {
		return s[:maxLen]
	}

	// Standard truncation
	return s[:maxLen-3] + "..."
}
