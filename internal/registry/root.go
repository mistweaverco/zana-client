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
	spinner     spinner.Model
	quitting    bool
	err         error
	message     string
	downloading bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	return model{spinner: s, message: "Getting latest info from the registry"}
}

func (m *model) Unzip() tea.Cmd {
	return func() tea.Msg {
		go func() {
			files.Unzip(files.GetTempPath()+files.PS+"zana-registry.json.zip", files.GetAppDataPath())
			// Send the message to notify that unzipping is finished
			tea.Println("") // Triggers UI update to process the message
		}()
		return unzipFinishedMsg{}
	}
}

func (m *model) downloadRegistry() (model, tea.Cmd) {
	m.message = "Downloading registry"
	m.downloading = true

	return *m, func() tea.Msg {
		done := make(chan struct{})
		go func() {
			files.Download(REGISTRY_URL, files.GetTempPath()+files.PS+"zana-registry.json.zip")
			done <- struct{}{}
		}()
		<-done // Wait for download to finish in the background goroutine
		return downloadFinishedMsg{}
	}
}

func (m model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if !m.downloading {
		return m.downloadRegistry()
	}
	switch msg := msg.(type) {
	case downloadFinishedMsg:
		m.message = "Registry downloaded successfully!"
		return m, m.Unzip()
	case unzipFinishedMsg:
		m.message = "Registry unzipped successfully!"
		m.quitting = true
		return m, tea.Quit
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
		var cmd tea.Cmd
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
