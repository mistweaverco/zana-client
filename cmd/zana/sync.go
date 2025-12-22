package zana

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh/spinner"
	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync registry or packages",
	Long: `Sync registry or packages.

The sync command has two subcommands:
  registry  - Download and unzip the latest registry file
  packages  - Ensure all packages in zana-lock.json are installed in exact versions`,
}

var syncRegistryCmd = &cobra.Command{
	Use:   "registry",
	Short: "Download and unzip the latest registry file",
	Long: `Download and unzip the latest registry file from the registry URL.

This command downloads the registry file and extracts it to the app data directory.
The registry URL can be overridden using the ZANA_REGISTRY_URL environment variable.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Downloading registry...")
		if err := syncRegistryFn(); err != nil {
			fmt.Printf("%s Failed to sync registry: %v\n", IconClose(), err)
			osExit(1)
			return
		}
		fmt.Printf("%s Registry synced successfully\n", IconCheck())
	},
}

var syncPackagesCmd = &cobra.Command{
	Use:   "packages",
	Short: "Sync all packages from zana-lock.json",
	Long: `Ensure all packages defined in zana-lock.json are installed in the exact versions specified.

This command reads the zana-lock.json file and ensures that all packages
are installed with their exact versions as specified in the lock file.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Syncing packages from zana-lock.json...")
		syncPackagesFn()
		fmt.Printf("%s Packages sync completed\n", IconCheck())
	},
}

func init() {
	syncCmd.AddCommand(syncRegistryCmd)
	syncCmd.AddCommand(syncPackagesCmd)
}

// downloadAndUnzipRegistryForced downloads and unzips the registry, forcing a fresh download
func downloadAndUnzipRegistryForced() error {
	registryURL := "https://github.com/mistweaverco/zana-registry/releases/latest/download/zana-registry.json.zip"
	if override := os.Getenv("ZANA_REGISTRY_URL"); override != "" {
		registryURL = override
	}

	cachePath := files.GetRegistryCachePath()

	// Force download by using 0 duration (cache is never valid) with spinner
	var downloadErr error
	action := func() {
		downloadErr = files.DownloadWithCache(registryURL, cachePath, 0)
	}

	if err := spinner.New().Title("Downloading registry...").Action(action).Run(); err != nil {
		return err
	}

	if downloadErr != nil {
		return fmt.Errorf("failed to download registry: %w", downloadErr)
	}

	// Unzip the registry
	if err := files.Unzip(cachePath, files.GetAppDataPath()); err != nil {
		return fmt.Errorf("failed to unzip registry: %w", err)
	}

	return nil
}

// indirections for testability
var (
	syncRegistryFn = downloadAndUnzipRegistryForced
	syncPackagesFn = providers.SyncAll
)
