package zana

import (
	"os"
	"runtime"

	"github.com/charmbracelet/log"
	"github.com/mistweaverco/zana-client/internal/config"
	"github.com/mistweaverco/zana-client/internal/registry"
	"github.com/mistweaverco/zana-client/internal/ui"
	"github.com/spf13/cobra"
)

var VERSION string
var cfg = config.NewConfig(config.Config{})

var rootCmd = &cobra.Command{
	Use:   "zana",
	Short: "Zana is Mason.nvim,  but maintained by the community",
	Long:  "A package manager for Neovim. Easily install and manage LSP servers, DAP servers, linters and formatters.",
	Run: func(cmd *cobra.Command, files []string) {
		if cfg.Flags.Version {
			log.Info("Version", runtime.GOOS, VERSION)
			return
		} else {
			registry.Update()
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
	rootCmd.PersistentFlags().BoolVar(&cfg.Flags.Version, "version", false, "version")
}
