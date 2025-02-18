package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
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

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

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
			lipgloss.Top,
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
	installedList  list.Model
	registryList   list.Model
	aboutPage      string
	tabs           []Tab
	activeTabIndex int
	searchInputs   []textinput.Model

	width, height  int
	spinner        spinner.Model
	spinnerVisible bool
	spinnerMessage string
	updating       bool
	updated        bool
	searchInput    textinput.Model
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if !m.updating {
		return m.initLists()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "shift+tab":
			m.activeTabIndex = (m.activeTabIndex - 1 + len(m.tabs)) % len(m.tabs)
			m.updateTabs()
		case "tab":
			m.activeTabIndex = (m.activeTabIndex + 1) % len(m.tabs)
			m.updateTabs()
		case "enter":
			switch m.getActiveTabId() {
			case "installed":
				selected := m.installedList.SelectedItem().(localPackageItem)
				if selected.updateAvailable {
					updater.Install(selected.sourceId, selected.version)
				}
			case "registry":
				selected := m.installedList.SelectedItem().(localPackageItem)
				updater.Install(selected.sourceId, selected.version)
			}
		case "/":
			m.searchInput.Focus()
			return m, m.searchInput.Focus()
		case "esc":
			m.searchInput.Blur()
			return m, nil
		}

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.width = msg.Width
		m.height = msg.Height
		m.installedList.SetSize(msg.Width-h, msg.Height-v-3) // Adjusted for tab height
		m.registryList.SetSize(msg.Width-h, msg.Height-v-3)  // Adjusted for tab height
	}

	if !m.spinnerVisible {
		switch m.getActiveTabId() {
		case "installed":
			m.installedList, cmd = m.installedList.Update(msg)
		case "registry":
			m.registryList, cmd = m.registryList.Update(msg)
		case "about":
		default:
		}
	} else {
		m.spinner, cmd = m.spinner.Update(msg)
	}

	// Handle search input updates
	if m.searchInput.Focused() {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, cmd
}

func (m *model) updateTabs() {
	for i := range m.tabs {
		m.tabs[i].IsActive = (i == m.activeTabIndex)
	}
}

func (m *model) getActiveTabId() string {
	for i := range m.tabs {
		if m.tabs[i].IsActive {
			return m.tabs[i].Id
		}
	}
	return ""
}

func (m model) View() string {
	doc := strings.Builder{}
	tabsRow := RenderTabs(m, m.tabs, m.width)
	doc.WriteString(tabsRow)

	if !m.spinnerVisible {
		switch m.getActiveTabId() {
		case "installed":
			doc.WriteString(docStyle.Render(m.installedList.View()))
		case "registry":
			doc.WriteString(docStyle.Render(m.registryList.View()))
		case "about":
			doc.WriteString(docStyle.Render(m.aboutPage))
		default:
		}
	} else {
		doc.WriteString(docStyle.Render(fmt.Sprintf("\n\n   %s "+m.spinnerMessage+"\n\n", m.spinner.View())))
	}

	return doc.String()
}

func (m model) setupRegistryList() list.Model {
	registryPackages := registry_parser.GetData(false)
	registryItems := []list.Item{}

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

	m.registryList.SetItems(registryItems)

	return m.registryList
}

func (m model) initLists() (tea.Model, tea.Cmd) {
	m.updating = true
	localPackages := local_packages_parser.GetData(false)
	installedItems := []list.Item{}

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

	m.installedList.SetItems(installedItems)

	m.registryList = m.setupRegistryList()

	m.spinnerMessage = "Updates checked"
	m.spinnerVisible = false
	m.updated = true

	return m, nil
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "Search..."
	ti.Width = 20
	ti.PromptStyle = lipgloss.NewStyle().Foreground(highlight)
	ti.Prompt = "🔍 "

	return model{
		installedList:  list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		registryList:   list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0),
		tabs:           []Tab{{Title: "Installed", IsActive: true, Id: "installed"}, {Title: "Registry", Id: "registry"}, {Title: "About", Id: "about"}},
		spinner:        spinner.New(spinner.WithSpinner(spinner.Points)),
		spinnerVisible: true,
		spinnerMessage: "Checking for updates",
		searchInput:    ti,
	}
}

func Show() {
	m := initialModel()

	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	m.installedList.SetShowTitle(false)
	m.registryList.SetShowTitle(false)

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
