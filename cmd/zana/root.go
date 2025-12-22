package zana

import (
	"fmt"
	"os"
	"time"

	"github.com/mistweaverco/zana-client/internal/config"
	"github.com/mistweaverco/zana-client/internal/lib/version"
	"github.com/spf13/cobra"
)

var cfg = config.NewConfig(config.Config{
	Flags: config.ConfigFlags{
		CacheMaxAge: 24 * time.Hour,       // Default to 24 hours
		Color:       config.ColorModeAuto, // Default to auto (respect TTY)
	},
})

var rootCmd = &cobra.Command{
	Use:   "zana",
	Short: "Zana is Mason.nvim, but not only for Neovim",
	Long:  "Zana is a minimal CLI for managing LSP servers, DAP servers, linters, and formatters, for Neovim, but not limited to just Neovim.",
	Run: func(cmd *cobra.Command, files []string) {
		if cfg.Flags.Version {
			fmt.Println(version.VERSION)
			return
		} else {
			// Show help if no command provided
			cmd.Help()
		}
	},
}

func Execute() {
	// Parse flags first to get color config
	err := rootCmd.Execute()
	if err != nil {
		osExit(1)
	}
}

// initColorConfig sets up the color config accessor for icons.go
// This must be called after flags are parsed
func initColorConfig() {
	// This function will be called from commands that use icons
	// We'll update the getColorConfig function in icons.go to return the actual config
}

func init() {
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.PersistentFlags().BoolVar(&cfg.Flags.Version, "version", false, "version")
	rootCmd.PersistentFlags().DurationVar(&cfg.Flags.CacheMaxAge, "cache-max-age", 24*time.Hour, "maximum age of registry cache (e.g., 1h, 24h, 7d)")
	colorFlag := rootCmd.PersistentFlags().VarPF(&cfg.Flags.Color, "color", "", "when to use colors and icons: always, auto (default), never")
	colorFlag.NoOptDefVal = string(config.ColorModeAlways) // If --color is used without value, default to "always"

	// Set up the color config accessor for icons.go
	SetColorConfigFunc(func() config.ConfigFlags {
		return cfg.Flags
	})
}

// osExit is a variable to allow overriding in tests
var osExit = os.Exit
