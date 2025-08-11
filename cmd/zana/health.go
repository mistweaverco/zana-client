package zana

import (
	"fmt"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check system health and requirements",
	Long: `Check if the system meets all requirements for running Zana.

This command verifies the presence of required tools and dependencies:
  - NPM (Node.js package manager)
  - Python interpreter
  - Python Distutils module
  - Go programming language
  - Cargo (Rust package manager)`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🔍 Checking system requirements...")
		fmt.Println()

		// Check requirements
		results := providers.CheckRequirements()

		// Display results
		displayRequirement("NPM", results.HasNPM, "Node.js package manager for JavaScript packages")
		displayRequirement("Python", results.HasPython, "Python interpreter for Python packages")
		displayRequirement("Python Distutils", results.HasPythonDistutils, "Python distutils module for building packages")
		displayRequirement("Go", results.HasGo, "Go programming language for Go packages")
		displayRequirement("Cargo", results.HasCargo, "Rust package manager for Rust packages")

		fmt.Println()

		// Overall status
		allMet := results.HasNPM && results.HasPython && results.HasPythonDistutils && results.HasGo && results.HasCargo
		if allMet {
			fmt.Println("✅ All requirements are met! Your system is ready to use Zana.")
		} else {
			fmt.Println("⚠️  Some requirements are not met. Some package managers may not work properly.")
			fmt.Println()
			fmt.Println("To install missing requirements:")
			fmt.Println("  - NPM: Install Node.js from https://nodejs.org/")
			fmt.Println("  - Python: Install Python from https://python.org/")
			fmt.Println("  - Go: Install Go from https://golang.org/")
			fmt.Println("  - Cargo: Install Rust/Cargo from https://rustup.rs/")
		}
	},
}

func displayRequirement(name string, met bool, description string) {
	var status, icon string
	if met {
		status = "Available"
		icon = "✅"
	} else {
		status = "Missing"
		icon = "❌"
	}

	fmt.Printf("%s %s: %s\n", icon, name, status)
	fmt.Printf("   %s\n", description)
	fmt.Println()
}
