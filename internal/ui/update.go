package ui

import (
	"log"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mistweaverco/zana-client/internal/lib/updater"
	"github.com/mistweaverco/zana-client/internal/modal"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Remove size limitations
		m.width = msg.Width
		m.height = msg.Height

		if m.modal != nil {
			*m.modal, _ = m.modal.Update(msg)
		}

		// Calculate dynamic column widths
		nameWidth := int(float64(m.width) * 0.7) // 70% of width for name
		versionWidth := m.width - nameWidth - 2  // Remaining width for version, accounting for borders

		// Ensure minimum widths
		minVersionWidth := 15
		if versionWidth < minVersionWidth {
			versionWidth = minVersionWidth
			nameWidth = m.width - versionWidth - 2
		}

		// Update column widths
		m.installedTable.SetColumns([]table.Column{
			{Title: "Name", Width: nameWidth},
			{Title: "Version", Width: versionWidth},
		})
		m.registryTable.SetColumns([]table.Column{
			{Title: "Name", Width: nameWidth},
			{Title: "Version", Width: versionWidth},
		})

		// Update table dimensions
		m.installedTable.SetWidth(m.width)
		m.registryTable.SetWidth(m.width)

		// Set height to leave room for tabs and borders
		tableHeight := m.height - 4 // Account for tabs and borders
		m.installedTable.SetHeight(tableHeight)
		m.registryTable.SetHeight(tableHeight)

		return m, nil

		// For all views
	case tea.KeyMsg:
		if m.modal != nil {
			updatedModal, cmd := m.modal.Update(msg)
			// Check for an empty message, which indicates a closed modal.
			if updatedModal.Message == "" {
				m.modal = nil
				return m, cmd
			}
			m.modal = &updatedModal
			return m, cmd
		}
		if !m.searchInput.Focused() {
			switch msg.String() {
			case "tab", "shift+tab", "right", "left", "h", "l":
				// Handle tab switching
				m.activeTabIndex = (m.activeTabIndex + 1) % len(m.tabs)
				// Update active state of tabs
				for i := range m.tabs {
					m.tabs[i].IsActive = (i == m.activeTabIndex)
				}
				m.currentView = m.tabs[m.activeTabIndex].Id
				return m, nil
			case "/":
				m.searchInput.Focus()
				return m, m.searchInput.Focus()
			case "esc":
				m.searchInput.Blur()
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}

			// Handle table navigation when search is not focused
			switch m.activeTabIndex {
			case 0:
				m.installedTable, cmd = m.installedTable.Update(msg)
			case 1:
				m.registryTable, cmd = m.registryTable.Update(msg)
			}
		} else {
			switch msg.String() {
			case "esc", "enter":
				m.searchInput.Blur()
				return m, nil
			}
			// Handle search input updates
			m.searchInput, cmd = m.searchInput.Update(msg)

			// Filter table rows based on search
			if m.activeTabIndex == 0 {
				m.filterInstalledTable(m.searchInput.Value())
			} else {
				m.filterRegistryTable(m.searchInput.Value())
			}
		}
	}

	// Only for specific views
	switch m.currentView {
	case "installed":
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if !m.searchInput.Focused() {
				switch msg.String() {
				case "i":
					// TODO: view package details
					return m.showModal("Not yet implemented", "warning")
				case "backspace":
					selectedIndex := m.installedTable.Cursor()
					data := getLocalPackagesData()
					row := data[selectedIndex]
					if updater.Remove(row.sourceId) == false {
						log.Println("Error uninstalling package")
						m.updateInstalledTableRows(getLocalPackagesData())
						return m.showModal("Error removing package", "error")
					} else {
						m.updateInstalledTableRows(getLocalPackagesData())
						return m.showModal("Package removed successfully", "success")
					}
				case "enter":
					selectedIndex := m.installedTable.Cursor()
					row := m.visibleInstalledData[selectedIndex]
					if updater.Install(row.sourceId, row.remoteVersion) == false {
						newModal := modal.New("Error installing package", "error")
						m.modal = &newModal
						log.Println("Error installing package")
						return m.showModal("Error installing package", "error")
					} else {
						return m.showModal("Package updated successfully", "success")
					}
				}
			}
		}
	case "registry":
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if !m.searchInput.Focused() {
				switch msg.String() {
				case "i":
					// TODO: view package details
					return m.showModal("Not yet implemented", "warning")
				case "enter":
					selectedIndex := m.registryTable.Cursor()
					row := m.visibleRegistryData[selectedIndex]
					if updater.Install(row.sourceId, row.version) == false {
						log.Println("Error installing package")
						return m.showModal("Error installing package", "error")
					}
					m.updateInstalledTableRows(getLocalPackagesData())
					return m.showModal("Package installed successfully", "success")
				}
			}
		}
	}

	return m, cmd
}
