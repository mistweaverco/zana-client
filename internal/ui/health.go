package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/providers"
)

type healthModel struct {
	list            list.Model
	results         providers.CheckRequirementsResult
	showingWarning  bool
	warningSelected int
}

type healthItem struct {
	title       string
	description string
	met         bool
}

func (i healthItem) Title() string       { return i.title }
func (i healthItem) Description() string { return i.description }
func (i healthItem) FilterValue() string { return i.title }

func getHealthTitle(name string, met bool) string {
	if met {
		return "✅ " + name
	}
	return "❌ " + name
}

func initialHealthModel(results providers.CheckRequirementsResult) healthModel {
	items := []list.Item{
		healthItem{title: getHealthTitle("NPM", results.HasNPM), description: "Node.js package manager for JavaScript packages", met: results.HasNPM},
		healthItem{title: getHealthTitle("Python", results.HasPython), description: "Python interpreter for Python packages", met: results.HasPython},
		healthItem{title: getHealthTitle("Python Distutils", results.HasPythonDistutils), description: "Python distutils module for building packages", met: results.HasPythonDistutils},
		healthItem{title: getHealthTitle("Go", results.HasGo), description: "Go programming language for Go packages", met: results.HasGo},
		healthItem{title: getHealthTitle("Cargo", results.HasCargo), description: "Rust package manager for Rust packages", met: results.HasCargo},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetShowHelp(true)
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	l.SetDelegate(delegate)

	return healthModel{list: l, results: results}
}

func (m healthModel) Init() tea.Cmd { return nil }

func (m healthModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			os.Exit(0)
		case "enter":
			if m.showingWarning {
				if m.warningSelected == 0 {
					return m, tea.Quit
				} else {
					os.Exit(0)
				}
			}
			if m.results.HasNPM && m.results.HasPython && m.results.HasPythonDistutils && m.results.HasGo && m.results.HasCargo {
				return m, tea.Quit
			}
			m.showingWarning = true
			return m, nil
		case "up", "k":
			if m.showingWarning && m.warningSelected > 0 {
				m.warningSelected--
			}
		case "down", "j":
			if m.showingWarning && m.warningSelected < 1 {
				m.warningSelected++
			}
		}
	case tea.WindowSizeMsg:
		h, v := docStyle.GetFrameSize()
		availableHeight := msg.Height - v - 8
		if availableHeight < 4 {
			availableHeight = 4
		}
		m.list.SetSize(msg.Width-h, availableHeight)
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m healthModel) View() string {
	if m.showingWarning {
		return m.renderWarningView()
	}
	allMet := m.results.HasNPM && m.results.HasPython && m.results.HasPythonDistutils && m.results.HasGo && m.results.HasCargo
	var status string
	if allMet {
		status = "✅ All requirements are met! Press Enter to continue."
	} else {
		status = "⚠️  Some requirements are not met."
	}
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true).Align(lipgloss.Center).MarginBottom(1)
	statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff00")).Bold(true).MarginBottom(1)
	if !allMet {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#ffaa00"))
	}
	title := titleStyle.Render("System Health Check")
	status = statusStyle.Render(status)
	listView := m.list.View()
	instructions := "Press Enter to continue or q to quit"
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", status, "", listView, "", instructions)
	return docStyle.Render(content)
}

func (m healthModel) renderWarningView() string {
	title := "⚠️  Warning: Some Requirements Not Met"
	var missingReqs []string
	if !m.results.HasNPM {
		missingReqs = append(missingReqs, "NPM")
	}
	if !m.results.HasPython {
		missingReqs = append(missingReqs, "Python")
	}
	if !m.results.HasPythonDistutils {
		missingReqs = append(missingReqs, "Python Distutils")
	}
	if !m.results.HasGo {
		missingReqs = append(missingReqs, "Go")
	}
	if !m.results.HasCargo {
		missingReqs = append(missingReqs, "Cargo")
	}
	missingText := "Missing requirements: " + strings.Join(missingReqs, ", ")
	options := []string{"Continue anyway (some features may not work)", "Quit and install missing requirements"}
	var optionTexts []string
	for i, option := range options {
		if i == m.warningSelected {
			optionTexts = append(optionTexts, "→ "+option)
		} else {
			optionTexts = append(optionTexts, "  "+option)
		}
	}
	content := lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ffaa00")).Bold(true).Render(title),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6666")).Render(missingText),
		"",
		"Some package managers may not work without these requirements.",
		"",
		strings.Join(optionTexts, "\n"),
		"",
		"Use ↑/↓ to navigate, Enter to select, q to quit",
	)
	return docStyle.Render(content)
}

func ShowHealthCheck() bool {
	results := providers.CheckRequirements()
	allMet := results.HasNPM && results.HasPython && results.HasPythonDistutils && results.HasGo && results.HasCargo
	if allMet {
		return true
	}
	m := initialHealthModel(results)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running health check:", err)
		os.Exit(1)
	}
	return true
}
