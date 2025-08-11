package zana

import (
	"os"
	"runtime"
	"time"

	"github.com/charmbracelet/log"
	"github.com/mistweaverco/zana-client/internal/boot"
	"github.com/mistweaverco/zana-client/internal/config"
	"github.com/mistweaverco/zana-client/internal/ui"
	"github.com/spf13/cobra"
)

var VERSION string
var cfg = config.NewConfig(config.Config{
	Flags: config.ConfigFlags{
		CacheMaxAge: 24 * time.Hour, // Default to 24 hours
	},
})

var rootCmd = &cobra.Command{
	Use:   "zana",
	Short: "Zana is Mason.nvim,  but maintained by the community",
	Long:  "A package manager for Neovim. Easily install and manage LSP servers, DAP servers, linters and formatters.",
	Run: func(cmd *cobra.Command, files []string) {
		if cfg.Flags.Version {
			log.Info("Version", runtime.GOOS, VERSION)
			return
		} else {
			// Check requirements before starting the main application
			if !ui.ShowRequirementsCheck() {
				log.Info("User chose to quit due to missing requirements")
				return
			}

			boot.Start(cfg.Flags.CacheMaxAge)
			ui.Show()
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(envCmd)
	rootCmd.PersistentFlags().BoolVar(&cfg.Flags.Version, "version", false, "version")
	rootCmd.PersistentFlags().DurationVar(&cfg.Flags.CacheMaxAge, "cache-max-age", 24*time.Hour, "maximum age of registry cache (e.g., 1h, 24h, 7d)")
}
