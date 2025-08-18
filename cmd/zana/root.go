package zana

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/log"
	"github.com/mistweaverco/zana-client/internal/boot"
	"github.com/mistweaverco/zana-client/internal/config"
	"github.com/mistweaverco/zana-client/internal/lib/version"
	"github.com/mistweaverco/zana-client/internal/ui"
	"github.com/spf13/cobra"
)

var cfg = config.NewConfig(config.Config{
	Flags: config.ConfigFlags{
		CacheMaxAge: 24 * time.Hour, // Default to 24 hours
	},
})

var rootCmd = &cobra.Command{
	Use:   "zana",
	Short: "Zana is Mason.nvim, but not only for Neovim",
	Long:  "Zana is a minimal CLI and TUI for managing LSP servers, DAP servers, linters, and formatters, for Neovim, but not limited to just Neovim.",
	Run: func(cmd *cobra.Command, files []string) {
		if cfg.Flags.Version {
			fmt.Println(version.VERSION)
			return
		} else {
			// Check health before starting the main application
			if !showHealthCheckFn() {
				log.Info("User chose to quit due to missing requirements")
				return
			}

			bootStartFn(cfg.Flags.CacheMaxAge)
			uiShowFn()
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		osExit(1)
	}
}

func init() {
	rootCmd.AddCommand(envCmd)
	rootCmd.AddCommand(healthCmd)
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.PersistentFlags().BoolVar(&cfg.Flags.Version, "version", false, "version")
	rootCmd.PersistentFlags().DurationVar(&cfg.Flags.CacheMaxAge, "cache-max-age", 24*time.Hour, "maximum age of registry cache (e.g., 1h, 24h, 7d)")
}

// osExit is a variable to allow overriding in tests
var osExit = os.Exit

// indirections for testability
var (
	showHealthCheckFn = ui.ShowHealthCheck
	bootStartFn       = boot.Start
	uiShowFn          = ui.Show
)
