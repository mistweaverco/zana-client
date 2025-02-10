package registry

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/files"
)

// REGISTRY_URL is the URL to the latest registry file
// use the environment variable ZANA_REGISTRY_URL to override
var DEFAULT_REGISTRY_URL = "https://github.com/mistweaverco/zana-registry/releases/latest/download/registry.json.zip"
var OVERRIDE_REGISTRY_URL = os.Getenv("ZANA_REGISTRY_URL")

var REGISTRY_URL = func() string {
	if OVERRIDE_REGISTRY_URL != "" {
		return OVERRIDE_REGISTRY_URL
	}
	return DEFAULT_REGISTRY_URL
}()

type errMsg error

type downloadFinishedMsg struct{}

type unzipFinishedMsg struct{}

type model struct {
	spinner          spinner.Model
	quitting         bool
	err              error
	message          string
	downloading      bool
	downloadFinished bool
	unzipFinished    bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{spinner: s, message: "Getting latest info from the registry"}
}

func (m model) Unzip() model {
	files.Unzip(files.GetTempPath()+files.PS+"zana-registry.json.zip", files.GetAppDataPath())
	m.message = "Registry unzipped successfully!"
	m.unzipFinished = true
	return m
}

func (m model) downloadRegistry() model {
	m.message = "Downloading registry"
	m.downloading = true
	files.Download(REGISTRY_URL, files.GetTempPath()+files.PS+"zana-registry.json.zip")
	m.message = "Registry downloaded successfully!"
	m.downloadFinished = true
	return m
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if !m.downloading {
		m.spinner, cmd = m.spinner.Update(msg)
		m = m.downloadRegistry()
	}
	if m.downloadFinished {
		m.spinner, cmd = m.spinner.Update(msg)
		m = m.Unzip()
	}
	if m.unzipFinished {
		m.spinner, cmd = m.spinner.Update(msg)
		m.quitting = true
		return m, tea.Quit
	}
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

	default:
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
}

func (m model) View() string {
	if m.err != nil {
		return m.err.Error()
	}
	str := fmt.Sprintf("\n\n   %s "+m.message+"\n\n", m.spinner.View())
	if m.quitting {
		return str + "\n"
	}
	return str
}

func Update() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
