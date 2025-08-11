package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
)

type requirementsModel struct {
	list            list.Model
	results         updater.CheckRequirementsResult
	selected        int
	showingWarning  bool
	warningSelected int
}

type requirementItem struct {
	title       string
	description string
	met         bool
}

func (i requirementItem) Title() string       { return i.title }
func (i requirementItem) Description() string { return i.description }
func (i requirementItem) FilterValue() string { return i.title }

// getRequirementTitle returns the title with appropriate icon
func getRequirementTitle(name string, met bool) string {
	if met {
		return "✅ " + name
	}
	return "❌ " + name
}

func initialRequirementsModel(results updater.CheckRequirementsResult) requirementsModel {
	items := []list.Item{
		requirementItem{
			title:       getRequirementTitle("NPM", results.HasNPM),
			description: "Node.js package manager for JavaScript packages",
			met:         results.HasNPM,
		},
		requirementItem{
			title:       getRequirementTitle("Python", results.HasPython),
			description: "Python interpreter for Python packages",
			met:         results.HasPython,
		},
		requirementItem{
			title:       getRequirementTitle("Python Distutils", results.HasPythonDistutils),
			description: "Python distutils module for building packages",
			met:         results.HasPythonDistutils,
		},
		requirementItem{
			title:       getRequirementTitle("Go", results.HasGo),
			description: "Go programming language for Go packages",
			met:         results.HasGo,
		},
	}

	l := list.New(items, list.NewDefaultDelegate(), 0, 0)
	l.SetShowHelp(true)

	// Customize the list delegate to show descriptions better
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	delegate.Styles.SelectedDesc = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	l.SetDelegate(delegate)

	return requirementsModel{
		list:    l,
		results: results,
	}
}

func (m requirementsModel) Init() tea.Cmd {
	return nil
}

func (m requirementsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			os.Exit(0)
		case "enter":
			if m.showingWarning {
				// Handle warning selection
				if m.warningSelected == 0 {
					// Continue anyway
					return m, tea.Quit
				} else {
					// Quit
					os.Exit(0)
				}
			}
			// Check if all requirements are met
			if m.results.HasNPM && m.results.HasPython &&
				m.results.HasPythonDistutils && m.results.HasGo {
				// All requirements met, continue to main app
				return m, tea.Quit
			}
			// Some requirements not met, show warning
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
		// Reserve space for title, status, and instructions
		availableHeight := msg.Height - v - 8 // 8 lines for title, status, and instructions
		if availableHeight < 4 {
			availableHeight = 4 // Minimum height for list
		}
		m.list.SetSize(msg.Width-h, availableHeight)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m requirementsModel) View() string {
	if m.showingWarning {
		return m.renderWarningView()
	}

	// Check if all requirements are met
	allMet := m.results.HasNPM && m.results.HasPython &&
		m.results.HasPythonDistutils && m.results.HasGo

	var status string
	if allMet {
		status = "✅ All requirements are met! Press Enter to continue."
	} else {
		status = "⚠️  Some requirements are not met."
	}

	// Style the title
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Align(lipgloss.Center).
		MarginBottom(1)

	// Style the status message
	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ff00")).
		Bold(true).
		MarginBottom(1)

	if !allMet {
		statusStyle = statusStyle.Foreground(lipgloss.Color("#ffaa00"))
	}

	// Build the layout with proper spacing
	title := titleStyle.Render("System Requirements Check")
	status = statusStyle.Render(status)
	listView := m.list.View()
	instructions := "Press Enter to continue or q to quit"
	// Join all elements with proper spacing
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		"",
		status,
		"",
		listView,
		"",
		instructions,
	)

	return docStyle.Render(content)
}

func (m requirementsModel) renderWarningView() string {
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

	missingText := "Missing requirements: " + strings.Join(missingReqs, ", ")

	options := []string{
		"Continue anyway (some features may not work)",
		"Quit and install missing requirements",
	}

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

func ShowRequirementsCheck() bool {
	// Check requirements first
	results := updater.CheckRequirements()
	allMet := results.HasNPM && results.HasPython &&
		results.HasPythonDistutils && results.HasGo

	if allMet {
		// All requirements met, proceed immediately
		return true
	}

	// Some requirements missing, show UI for user choice
	m := initialRequirementsModel(results)
	p := tea.NewProgram(m)

	// Run the requirements check UI
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running requirements check:", err)
		os.Exit(1)
	}

	// Show warning and let user decide
	return true
}

type warningModel struct {
	results        updater.CheckRequirementsResult
	shouldContinue bool
	selected       int
}

func (m warningModel) Init() tea.Cmd {
	return nil
}

func (m warningModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "q", "ctrl+c":
			os.Exit(0)
		case "enter":
			if m.selected == 0 {
				// Continue anyway
				m.shouldContinue = true
				return m, tea.Quit
			} else {
				// Quit
				os.Exit(0)
			}
		case "up", "k":
			if m.selected > 0 {
				m.selected--
			}
		case "down", "j":
			if m.selected < 1 {
				m.selected++
			}
		}
	}

	return m, nil
}

func (m warningModel) View() string {
	title := "⚠️  Warning: Some Requirements Not Met. Press enter to continue or q to quit."

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

	missingText := "Missing requirements: " + strings.Join(missingReqs, ", ")

	options := []string{
		"Continue anyway (some features may not work)",
		"Quit and install missing requirements",
	}

	var optionTexts []string
	for i, option := range options {
		if i == m.selected {
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