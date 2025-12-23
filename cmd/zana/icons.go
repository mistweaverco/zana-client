package zana

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/mistweaverco/zana-client/internal/config"
)

// Nerd Font icons
// Using Font Awesome and Devicons which are stable and widely supported
// Codepoints verified for Nerd Fonts v3.0+
// Font Awesome range: U+E000-U+F8FF (Private Use Area)
// Devicons range: U+E000-U+E7FF

// getColorConfigFunc provides access to the color config from root.go
// This is set in root.go's init() function
var getColorConfigFunc func() config.ConfigFlags

// setColorConfigFunc sets the function to access color config
func SetColorConfigFunc(fn func() config.ConfigFlags) {
	getColorConfigFunc = fn
}

// getColorConfig returns the current config instance
func getColorConfig() config.ConfigFlags {
	if getColorConfigFunc != nil {
		return getColorConfigFunc()
	}
	// Default implementation if not set
	return config.ConfigFlags{Color: config.ColorModeAuto}
}

// shouldUseColors determines if colors/icons should be used based on color mode and TTY status
func shouldUseColors() bool {
	colorMode := getColorConfig().Color
	isTTY := isatty.IsTerminal(os.Stdout.Fd())

	switch colorMode {
	case config.ColorModeAlways:
		return true
	case config.ColorModeNever:
		return false
	case config.ColorModeAuto:
		fallthrough
	default:
		return isTTY
	}
}

// ANSI color codes
const (
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"

	// Bright colors
	colorBrightRed     = "\033[91m"
	colorBrightGreen   = "\033[92m"
	colorBrightYellow  = "\033[93m"
	colorBrightBlue    = "\033[94m"
	colorBrightMagenta = "\033[95m"
	colorBrightCyan    = "\033[96m"
)

// Base icon constants (uncolored)
const (
	// Status icons (Font Awesome - solid style)
	iconCheck       = "\uF00C" // nf-fa-check (U+F00C)
	iconClose       = "\uF00D" // nf-fa-times (U+F00D)
	iconCheckCircle = "\uF058" // nf-fa-check_circle (U+F058)
	iconCancel      = "\uF057" // nf-fa-times_circle (U+F057)

	// Action icons (Font Awesome)
	iconMagnify   = "\uF002" // nf-fa-search (U+F002)
	iconAlert     = "\uF071" // nf-fa-exclamation_triangle (U+F071)
	iconRefresh   = "\uF021" // nf-fa-refresh (U+F021)
	iconLightbulb = "\uF0EB" // nf-fa-lightbulb_o (U+F0EB)
	iconSummary   = "\uF080" // nf-fa-bar_chart (U+F080)

	// Provider icons (Devicons / Font Awesome)
	// NOTE: Icons marked with "TODO" don't have specific Nerd Font icons yet and are using fallbacks
	iconNPM      = "\uE71E"     // nf-dev-npm (U+E71E)
	iconGolang   = "\uE627"     // nf-dev-go (U+E627)
	iconPython   = "\uE63C"     // nf-dev-python (U+E63C)
	iconCargo    = "\uE7A8"     // nf-dev-rust (U+E7A8)
	iconGitHub   = "\uF09B"     // nf-fa-github (U+F09B)
	iconGitLab   = "\uE65D"     // nf-dev-gitlab (U+E65D)
	iconCodeberg = "\U000F1A9F" // nf-md-mountain (U+F1A9F) - using mountain icon for Codeberg
	iconGem      = "\uE739"     // nf-dev-ruby (U+E739)
	iconComposer = "\uE7D2"     // nf-dev-php (U+E7D2) - using PHP icon for Composer
	iconLuaRocks = "\uE620"     // nf-dev-lua (U+E620) - using Lua icon for LuaRocks
	iconNuGet    = "\uE77E"     // nf-dev-csharp (U+E77E) - using C# icon for NuGet
	iconOpam     = "\uE7A1"     // nf-dev-ocaml (U+E7A1)
	iconOpenVSX  = "\uE7C5"     // TODO: Using Visual Studio icon, no specific OpenVSX icon in Nerd Fonts
	iconGeneric  = "\uF1C6"     // nf-fa-archive (U+F1C6)
)

// Plain text alternatives for icons when not in TTY
const (
	textCheck       = "[✓]"
	textClose       = "[✗]"
	textCheckCircle = "[✓]"
	textCancel      = "[✗]"
	textAlert       = "[!]"
	textMagnify     = "[?]"
	textRefresh     = "[~]"
	textLightbulb   = "[*]"
	textSummary     = "[=]"
	textNPM         = "[npm]"
	textGolang      = "[go]"
	textPython      = "[py]"
	textCargo       = "[rs]"
	textGitHub      = "[gh]"
	textGitLab      = "[gl]"
	textCodeberg    = "[cb]"
	textGem         = "[rb]"
	textComposer    = "[php]"
	textLuaRocks    = "[lua]"
	textNuGet       = "[cs]"
	textOpam        = "[ocaml]"
	textOpenVSX     = "[vsx]"
	textGeneric     = "[pkg]"
)

// Colored icon functions
// When output is piped (not a TTY) or color mode is 'never', return plain text alternatives without colors
// Success icons (green)
func IconCheck() string {
	if !shouldUseColors() {
		return textCheck
	}
	return colorGreen + iconCheck + colorReset
}

func IconCheckCircle() string {
	if !shouldUseColors() {
		return textCheckCircle
	}
	return colorGreen + iconCheckCircle + colorReset
}

// Error icons (red)
func IconClose() string {
	if !shouldUseColors() {
		return textClose
	}
	return colorRed + iconClose + colorReset
}

func IconCancel() string {
	if !shouldUseColors() {
		return textCancel
	}
	return colorRed + iconCancel + colorReset
}

// Warning icons (yellow)
func IconAlert() string {
	if !shouldUseColors() {
		return textAlert
	}
	return colorYellow + iconAlert + colorReset
}

// Info icons (cyan/blue)
func IconMagnify() string {
	if !shouldUseColors() {
		return textMagnify
	}
	return colorCyan + iconMagnify + colorReset
}

func IconRefresh() string {
	if !shouldUseColors() {
		return textRefresh
	}
	return colorCyan + iconRefresh + colorReset
}

func IconLightbulb() string {
	if !shouldUseColors() {
		return textLightbulb
	}
	return colorYellow + iconLightbulb + colorReset
}

func IconSummary() string {
	if !shouldUseColors() {
		return textSummary
	}
	return colorBlue + iconSummary + colorReset
}

// Provider icons with brand colors
func IconNPM() string {
	if !shouldUseColors() {
		return textNPM
	}
	return colorRed + iconNPM + colorReset
}

func IconGolang() string {
	if !shouldUseColors() {
		return textGolang
	}
	return colorCyan + iconGolang + colorReset
}

func IconPython() string {
	if !shouldUseColors() {
		return textPython
	}
	return colorGreen + iconPython + colorReset
}

func IconCargo() string {
	if !shouldUseColors() {
		return textCargo
	}
	return colorRed + iconCargo + colorReset
}

func IconGitHub() string {
	if !shouldUseColors() {
		return textGitHub
	}
	return colorWhite + iconGitHub + colorReset
}

func IconGitLab() string {
	if !shouldUseColors() {
		return textGitLab
	}
	return colorMagenta + iconGitLab + colorReset
}

func IconCodeberg() string {
	if !shouldUseColors() {
		return textCodeberg
	}
	return colorCyan + iconCodeberg + colorReset // Mountain in cyan
}

func IconGem() string {
	if !shouldUseColors() {
		return textGem
	}
	return colorRed + iconGem + colorReset
}

func IconComposer() string {
	if !shouldUseColors() {
		return textComposer
	}
	return colorBlue + iconComposer + colorReset
}

func IconLuaRocks() string {
	if !shouldUseColors() {
		return textLuaRocks
	}
	return colorBlue + iconLuaRocks + colorReset
}

func IconNuGet() string {
	if !shouldUseColors() {
		return textNuGet
	}
	return colorMagenta + iconNuGet + colorReset
}

func IconOpam() string {
	if !shouldUseColors() {
		return textOpam
	}
	return colorYellow + iconOpam + colorReset
}

func IconOpenVSX() string {
	if !shouldUseColors() {
		return textOpenVSX
	}
	return colorBlue + iconOpenVSX + colorReset
}

func IconGeneric() string {
	if !shouldUseColors() {
		return textGeneric
	}
	return colorWhite + iconGeneric + colorReset
}
