package boot

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
)

var DEFAULT_REGISTRY_URL = "https://github.com/mistweaverco/zana-registry/releases/latest/download/registry.json.zip"

type errMsg error

type downloadStartedMsg struct{}
type downloadFinishedMsg struct{}
type unzipStartedMsg struct{}
type unzipFinishedMsg struct{}
type syncLocalPackagesStartedMsg struct{}
type syncLocalPackagesFinishedMsg struct{}

type model struct {
	spinner          spinner.Model
	quitting         bool
	err              error
	message          string
	downloading      bool
	downloadFinished bool
	registryURL      string
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	registryURL := DEFAULT_REGISTRY_URL
	override := os.Getenv("ZANA_REGISTRY_URL")
	if override != "" {
		registryURL = override
	}
	return model{spinner: s, message: "Getting latest info from the registry", registryURL: registryURL}
}

func (m model) downloadRegistry() tea.Cmd {
	return func() tea.Msg {
		return downloadStartedMsg{}
	}
}

func (m model) performDownload() tea.Cmd {
	return func() tea.Msg {
		files.Download(m.registryURL, files.GetTempPath()+files.PS+"zana-registry.json.zip")
		return downloadFinishedMsg{}
	}
}

func (m model) unzipRegistry() tea.Cmd {
	return func() tea.Msg {
		return unzipStartedMsg{}
	}
}

func (m model) performUnzip() tea.Cmd {
	return func() tea.Msg {
		files.Unzip(files.GetTempPath()+files.PS+"zana-registry.json.zip", files.GetAppDataPath())
		return unzipFinishedMsg{}
	}
}

func (m model) syncLocalPackages() tea.Cmd {
	return func() tea.Msg {
		return syncLocalPackagesStartedMsg{}
	}
}

func (m model) performSyncLocalPackages() tea.Cmd {
	return func() tea.Msg {
		updater.SyncAll()
		return syncLocalPackagesFinishedMsg{}
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.downloadRegistry())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		default:
			return m, nil
		}

	case errMsg:
		m.err = msg
		return m, nil

	case downloadStartedMsg:
		m.message = "Downloading registry"
		m.downloading = true
		return m, m.performDownload()

	case downloadFinishedMsg:
		m.message = "Registry downloaded successfully!"
		m.downloadFinished = true
		return m, m.unzipRegistry()

	case unzipStartedMsg:
		m.message = "Unzipping registry"
		return m, m.performUnzip()

	case unzipFinishedMsg:
		m.message = "Registry unzipped successfully!"
		return m, m.syncLocalPackages()

	case syncLocalPackagesStartedMsg:
		m.message = "Syncing local packages"
		return m, m.performSyncLocalPackages()

	case syncLocalPackagesFinishedMsg:
		m.message = "Local packages synced successfully!"
		m.quitting = true
		return m, tea.Quit

	default:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	str := fmt.Sprintf("\n\n  %s "+m.message+"\n\n", m.spinner.View())
	if m.quitting {
		return str + "\n"
	}
	return str
}

func Start() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
