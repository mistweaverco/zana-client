package modal

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Define key mappings
type keyMap struct {
	Quit  key.Binding
	Close key.Binding
}

var keys = keyMap{
	Quit:  key.NewBinding(key.WithKeys("ctrl+c")),
	Close: key.NewBinding(key.WithKeys("esc", "enter")),
}

// BubbleTea component for the modal
type Modal struct {
	Message  string // Exported Message field
	width    int
	height   int
	quitting bool
	keys     keyMap
}

func New(msg string) Modal {
	return Modal{
		Message: msg,
		keys:    keys,
	}
}

func (m Modal) Update(msg tea.Msg) (Modal, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Update width and height
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, m.keys.Close) {
			return Modal{}, nil // Returning an empty modal closes it.
		}
		if key.Matches(msg, m.keys.Quit) {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil
	}

	return m, nil
}

func (m Modal) View() string {
	return m.view(m.width, m.height)
}

func (m Modal) view(screenWidth, screenHeight int) string {
	// Calculate the width and height of the modal based on message length
	maxLineWidth := 0
	lines := strings.Split(m.Message, "\n")
	for _, line := range lines {
		if len(line) > maxLineWidth {
			maxLineWidth = len(line)
		}
	}

	modalWidth := maxLineWidth + 4 // Add padding
	modalHeight := len(lines) + 4  // Add padding and button

	// Limit modal size to screen size
	if modalWidth > screenWidth-4 {
		modalWidth = screenWidth - 4
	}
	if modalHeight > screenHeight-4 {
		modalHeight = screenHeight - 4
	}

	// Create the modal content
	content := lipgloss.NewStyle().Width(modalWidth - 4).Align(lipgloss.Center).Render(m.Message)
	closeButton := lipgloss.NewStyle().Padding(0, 1).Render("[Close]")

	// Create the modal box
	modalBox := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Center, content, closeButton))

	// Position the modal box
	return lipgloss.Place(screenWidth, screenHeight, lipgloss.Center, lipgloss.Center, modalBox)
}
