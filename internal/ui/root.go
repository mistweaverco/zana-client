package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
)

type updateCheckFinishedMsg struct{}

var (
	// General styles
	normal    = lipgloss.Color("#EEEEEE")
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	base = lipgloss.NewStyle().Foreground(normal)

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	updateAvailableStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
	missingInRegistryStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	checkingForUpdatesStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#73F59F"))
	installedVersionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

// Tab struct for modular tabs
type Tab struct {
	Title    string
	IsActive bool
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
func RenderTabs(tabs []Tab, totalWidth int) string {
	var renderedTabs []string
	for _, tab := range tabs {
		renderedTabs = append(renderedTabs, tab.Render())
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	gapWidth := totalWidth - lipgloss.Width(row)
	if gapWidth > 0 {
		gap := strings.Repeat("─", gapWidth)
		row += lipgloss.NewStyle().Foreground(highlight).Render(gap)
	}

	return row
}

// Item struct for the list
type item struct {
	title, desc, sourceId string
	updateAvailable       bool
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Main model
type model struct {
	installedList  list.Model
	tabs           []Tab
	activeTabIndex int

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
	if !m.updating {
		return m.fetchUpdates()
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.activeTabIndex = (m.activeTabIndex + 1) % len(m.tabs)
			m.updateTabs()
		}

	case updateCheckFinishedMsg:
		m.updating = false
		m.spinnerVisible = false
		m.installedList, cmd = m.installedList.Update(msg)
		return m, cmd

	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.width = msg.Width
		m.height = msg.Height
		m.installedList.SetSize(msg.Width-h, msg.Height-v-3) // Adjusted for tab height
	}

	if !m.spinnerVisible {
		m.installedList, cmd = m.installedList.Update(msg)
	} else {
		m.spinner, cmd = m.spinner.Update(msg)
	}

	return m, cmd
}

func (m *model) updateTabs() {
	for i := range m.tabs {
		m.tabs[i].IsActive = (i == m.activeTabIndex)
	}
}

func (m model) View() string {
	doc := strings.Builder{}
	doc.WriteString(RenderTabs(m.tabs, m.width))

	if !m.spinnerVisible {
		doc.WriteString(docStyle.Render(m.installedList.View()))
	} else {
		doc.WriteString(docStyle.Render(fmt.Sprintf("\n\n   %s "+m.spinnerMessage+"\n\n", m.spinner.View())))
	}

	return doc.String()
}

func (m model) fetchUpdates() (tea.Model, tea.Cmd) {
	m.updating = true
	localPackages := local_packages_parser.GetData(false)
	items := []list.Item{}

	for _, localPackage := range localPackages.Packages {
		regItem := registry_parser.GetBySourceId(localPackage.SourceID)
		updateAvailable, remoteVersion := updater.CheckIfUpdateIsAvailable(localPackage.Version, regItem.Source.ID)

		localItem := item{
			sourceId:        localPackage.SourceID,
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

		items = append(items, localItem)
	}

	m.installedList.SetItems(items)
	m.spinnerMessage = "Updates checked"
	m.spinnerVisible = false
	m.updated = true

	return m, nil
}

func Show() {
	m := model{
		spinner:        spinner.New(),
		spinnerVisible: true,
		spinnerMessage: "Checking for updates",
		tabs: []Tab{
			{Title: "Installed", IsActive: true},
			{Title: "Search Registry"},
			{Title: "About"},
		},
	}

	m.spinner.Spinner = spinner.Dot
	m.spinner.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	m.installedList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.installedList.SetShowTitle(false)

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
