package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
)

var (

	// General.

	normal    = lipgloss.Color("#EEEEEE")
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	base = lipgloss.NewStyle().Foreground(normal)

	divider = lipgloss.NewStyle().
		SetString("•").
		Padding(0, 1).
		Foreground(subtle).
		String()

	url = lipgloss.NewStyle().Foreground(special).Render

	// Tabs.

	activeTabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      " ",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┘",
		BottomRight: "└",
	}

	tabBorder = lipgloss.Border{
		Top:         "─",
		Bottom:      "─",
		Left:        "│",
		Right:       "│",
		TopLeft:     "╭",
		TopRight:    "╮",
		BottomLeft:  "┴",
		BottomRight: "┴",
	}

	tab = lipgloss.NewStyle().
		Border(tabBorder, true).
		BorderForeground(highlight).
		Padding(0, 1)

	activeTab = tab.Border(activeTabBorder, true)

	tabGap = tab.
		BorderTop(false).
		BorderLeft(false).
		BorderRight(false)

	// Page.

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)

	updateAvailableStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00"))
	missingInRegistryStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))
	checkingForUpdatesStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#73F59F"))
	installedVersionStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444"))
)

type item struct {
	title, desc, sourceId string
	updateAvailable       bool
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type model struct {
	installedList list.Model
	tabs          list.Model
	width, height int
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		m.width = msg.Width
		m.height = msg.Height
		tabHeight, _ := activeTab.GetFrameSize()
		m.installedList.SetSize(msg.Width-h, msg.Height-v-tabHeight)
	}

	var cmd tea.Cmd
	m.installedList, cmd = m.installedList.Update(msg)
	return m, cmd
}

func (m model) View() string {
	doc := strings.Builder{}
	row := lipgloss.JoinHorizontal(
		lipgloss.Top,
		activeTab.Render("Installed"),
		tab.Render("Search Registry"),
		tab.Render("About"),
	)
	gap := tabGap.Render(strings.Repeat(" ", max(0, m.width-lipgloss.Width(row)-2)))
	row = lipgloss.JoinHorizontal(lipgloss.Bottom, row, gap)
	doc.WriteString(row)
	doc.WriteString(docStyle.Render(m.installedList.View()))
	return doc.String()
}

func Show() {

	localPackages := local_packages_parser.GetData(false)

	items := []list.Item{}

	for _, localPackage := range localPackages.Packages {
		regItem := registry_parser.GetBySourceId(localPackage.SourceID)
		updateAvailable, remoteVersion := updater.CheckIfUpdateIsAvailable(localPackage.Version, regItem.Source.ID)
		localItem := item{
			sourceId:        localPackage.SourceID,
			updateAvailable: updateAvailable,
		}
		// Not found in registry,
		// So we could check for updates, but can't install it,
		// because we have no information on how to install it.
		if regItem.Source.ID == "" {
			localItem.title = localPackage.SourceID
			localItem.desc = installedVersionStyle.Render(localPackage.Version) + " " + missingInRegistryStyle.Render("Not found in registry")
		} else if updateAvailable {
			localItem.title = regItem.Name
			localItem.desc = regItem.Description + " " + installedVersionStyle.Render(localPackage.Version) + " " + updateAvailableStyle.Render("Update available: "+remoteVersion)
		} else if remoteVersion == "" {
			localItem.title = regItem.Name
			localItem.desc = regItem.Description + " " + installedVersionStyle.Render(localPackage.Version) + " " + missingInRegistryStyle.Render("No remote version found")
		} else {
			localItem.title = regItem.Name
			localItem.desc = regItem.Description + " " + installedVersionStyle.Render(localPackage.Version)
		}
		items = append(items, localItem)
	}

	m := model{
		installedList: list.New(items, list.NewDefaultDelegate(), 0, 0),
	}

	m.installedList.SetShowTitle(false)

	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
