package zana

import (
	"os"

	"github.com/mattn/go-isatty"
	"github.com/mistweaverco/zana-client/internal/config"
)

// Emoji icons
// Centralized location for all emoji icons used throughout the application

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
	// Status icons
	iconCheck       = "✓"
	iconClose       = "✗"
	iconCheckCircle = "✅"
	iconCancel      = "❌"

	// Action icons
	iconMagnify   = "🔍"
	iconAlert     = "⚠️"
	iconRefresh   = "🔄"
	iconLightbulb = "💡"
	iconSummary   = "📊"

	// Provider icons
	iconNPM      = "📦"
	iconGolang   = "🐹"
	iconPython   = "🐍"
	iconCargo    = "🦀"
	iconGitHub   = "🐙"
	iconGitLab   = "🦊"
	iconCodeberg = "🏔️"
	iconGem      = "💎"
	iconComposer = "🐘"
	iconLuaRocks = "🌙"
	iconNuGet    = "📦"
	iconOpam     = "🐫"
	iconOpenVSX  = "🔌"
	iconGeneric  = "📦"
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
