package zana

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mistweaverco/zana-client/internal/lib/providers"
	"github.com/mistweaverco/zana-client/internal/lib/semver"
	"github.com/mistweaverco/zana-client/internal/lib/version"
	"github.com/spf13/cobra"
)

// UpdateService handles update operations with dependency injection
type UpdateService struct {
	localPackages LocalPackagesProvider
	output        OutputWriter
}

// OutputWriter defines the interface for writing output (for testing)
type OutputWriter interface {
	Println(args ...interface{})
	Printf(format string, args ...interface{})
}

// DefaultOutputWriter implements OutputWriter using fmt
type DefaultOutputWriter struct{}

func (d *DefaultOutputWriter) Println(args ...interface{}) {
	fmt.Println(args...)
}

func (d *DefaultOutputWriter) Printf(format string, args ...interface{}) {
	fmt.Printf(format, args...)
}

// Mock implementations for testing
type MockPackageManager struct {
	UpdateFunc              func(sourceID string) bool
	IsSupportedProviderFunc func(provider string) bool
	AvailableProvidersFunc  func() []string
}

func (m *MockPackageManager) Update(sourceID string) bool {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(sourceID)
	}
	return false
}

func (m *MockPackageManager) IsSupportedProvider(provider string) bool {
	if m.IsSupportedProviderFunc != nil {
		return m.IsSupportedProviderFunc(provider)
	}
	return false
}

func (m *MockPackageManager) AvailableProviders() []string {
	if m.AvailableProvidersFunc != nil {
		return m.AvailableProvidersFunc()
	}
	return []string{}
}

// MockOutputWriter is a mock implementation for testing
type MockOutputWriter struct {
	PrintlnFunc func(args ...interface{})
	PrintfFunc  func(format string, args ...interface{})
	Output      []string
}

func (m *MockOutputWriter) Println(args ...interface{}) {
	if m.PrintlnFunc != nil {
		m.PrintlnFunc(args...)
	}
	// Capture output for testing
	if len(args) > 0 {
		m.Output = append(m.Output, fmt.Sprint(args...))
	}
}

func (m *MockOutputWriter) Printf(format string, args ...interface{}) {
	if m.PrintfFunc != nil {
		m.PrintfFunc(format, args...)
	}
	// Capture output for testing
	m.Output = append(m.Output, fmt.Sprintf(format, args...))
}

// NewUpdateService creates a new UpdateService with default dependencies
func NewUpdateService() *UpdateService {
	return &UpdateService{
		localPackages: &defaultLocalPackagesProvider{},
		output:        &DefaultOutputWriter{},
	}
}

// NewUpdateServiceWithDependencies creates a new UpdateService with custom dependencies
func NewUpdateServiceWithDependencies(
	localPackages LocalPackagesProvider,
	output OutputWriter,
) *UpdateService {
	return &UpdateService{
		localPackages: localPackages,
		output:        output,
	}
}

// updatePackage updates a single package using the provider factory system
func (us *UpdateService) updatePackage(sourceID string) bool {
	// Use the provider factory system which can be mocked in tests
	return providers.Update(sourceID)
}

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"up"},
	Short:   "Update packages to their latest versions",
	Long: `Update packages to their latest versions.

Examples:
  zana update npm:eslint
  zana update golang:golang.org/x/tools/gopls npm:prettier
  zana update pypi:black cargo:ripgrep
  zana update --all (update all installed packages)
  zana update --self (update zana itself to the latest version)`,
	Args: cobra.MinimumNArgs(0), // Allow no args if --all or --self is used
	// Enable shell completion for installed package IDs only.
	ValidArgsFunction: installedPackageIDCompletion,
	Run: func(cmd *cobra.Command, args []string) {
		selfFlag, _ := cmd.Flags().GetBool("self")
		if selfFlag {
			service := newUpdateService()
			if err := runSelfUpdate(service.output); err != nil {
				service.output.Printf("✗ Failed to update zana: %v\n", err)
				osExit(1)
				return
			}
			return
		}

		allFlag, _ := cmd.Flags().GetBool("all")

		if allFlag {
			// Update all installed packages
			service := newUpdateService()
			service.output.Println("Updating all installed packages to latest versions...")

			success := service.UpdateAllPackages()

			if success {
				service.output.Println("Successfully updated all packages")
			} else {
				service.output.Println("Failed to update some packages")
			}
			return
		}

		// Check if package IDs are provided
		if len(args) == 0 {
			service := newUpdateService()
			service.output.Println("Error: Please provide package IDs or use --all flag")
			return
		}

		// Process all packages
		packages := args
		internalIDs := make([]string, len(packages))
		displayIDs := make([]string, len(packages))

		for i, userPkgID := range packages {
			// Parse package ID and version from the user-facing ID
			baseID, _ := parsePackageIDAndVersion(userPkgID)

			var internalID string
			var displayID string

			// Check if this is a package name without provider
			if !strings.Contains(baseID, ":") && !strings.HasPrefix(baseID, "pkg:") {
				// Package name without provider - search installed packages and prompt user
				matches := findInstalledPackagesByName(baseID)
				if len(matches) == 0 {
					service := newUpdateService()
					service.output.Printf("✗ No installed packages found matching '%s'\n", baseID)
					return
				}

				selectedSourceID, err := promptForProviderSelection(baseID, matches)
				if err != nil {
					service := newUpdateService()
					service.output.Printf("✗ Error selecting provider for '%s': %v\n", baseID, err)
					return
				}

				internalID = selectedSourceID
				displayID = userPkgID
			} else {
				// Package with provider - parse normally
				provider, pkgName, err := parseUserPackageID(baseID)
				if err != nil {
					service := newUpdateService()
					service.output.Printf("Error: %v\n", err)
					return
				}
				if !providers.IsSupportedProvider(provider) {
					service := newUpdateService()
					service.output.Printf("Error: Unsupported provider '%s' for package '%s'. Supported providers: %s\n", provider, userPkgID, strings.Join(providers.AvailableProviders, ", "))
					return
				}

				internalID = toInternalPackageID(provider, pkgName)
				displayID = userPkgID
			}

			internalIDs[i] = internalID
			displayIDs[i] = displayID
		}

		// Update individual packages
		service := newUpdateService()
		service.output.Printf("Updating %d package(s) to latest versions...\n", len(packages))

		allSuccess := true
		successCount := 0
		failedCount := 0

		for idx := range packages {
			internalID := internalIDs[idx]
			displayID := displayIDs[idx]
			service.output.Printf("Updating %s...\n", displayID)

			// Update the package using the service method (which can be mocked in tests)
			success := service.updatePackage(internalID)
			if success {
				service.output.Printf("✓ Successfully updated %s\n", displayID)
				successCount++
			} else {
				service.output.Printf("✗ Failed to update %s\n", displayID)
				failedCount++
				allSuccess = false
			}
		}

		// Print summary
		service.output.Printf("\nUpdate Summary:\n")
		service.output.Printf("  Successfully updated: %d\n", successCount)
		service.output.Printf("  Failed to update: %d\n", failedCount)

		if allSuccess {
			service.output.Printf("All packages updated successfully!\n")
		} else {
			service.output.Printf("Some packages failed to update.\n")
		}
	},
}

func init() {
	updateCmd.Flags().BoolP("all", "A", false, "Update all installed packages to their latest versions")
	updateCmd.Flags().Bool("self", false, "Update zana itself to the latest version")
}

// newUpdateService is a factory to allow test injection
var newUpdateService = NewUpdateService

// UpdateAllPackages updates all installed packages to their latest versions
func (us *UpdateService) UpdateAllPackages() bool {
	// Get all installed packages
	localPackages := us.localPackages.GetData(true).Packages

	if len(localPackages) == 0 {
		us.output.Println("No packages are currently installed")
		return true
	}

	us.output.Printf("Found %d installed packages\n", len(localPackages))

	allSuccess := true
	successCount := 0
	failedCount := 0

	for _, pkg := range localPackages {
		us.output.Printf("Updating %s...\n", pkg.SourceID)

		// Use the service method (which can be mocked in tests)
		success := us.updatePackage(pkg.SourceID)
		if success {
			successCount++
			us.output.Printf("✓ Successfully updated %s\n", pkg.SourceID)
		} else {
			failedCount++
			us.output.Printf("✗ Failed to update %s\n", pkg.SourceID)
			allSuccess = false
		}
	}

	us.output.Printf("\nUpdate Summary:\n")
	us.output.Printf("  Successfully updated: %d\n", successCount)
	us.output.Printf("  Failed to update: %d\n", failedCount)

	return allSuccess
}

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// getCurrentVersion returns the current version of zana
func getCurrentVersion() string {
	return version.VERSION
}

// getLatestVersion fetches the latest release version from GitHub
func getLatestVersion() (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Get the latest release
	resp, err := client.Get("https://api.github.com/repos/mistweaverco/zana-client/releases/latest")
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var release GitHubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return "", fmt.Errorf("failed to parse release data: %w", err)
	}

	return release.TagName, nil
}

// getCurrentBinaryPath returns the path to the current zana binary
func getCurrentBinaryPath() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks to get the actual file path
	resolvedPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		// If symlink resolution fails, use the original path
		resolvedPath = execPath
	}

	return resolvedPath, nil
}

// detectPlatform returns the platform string for the current system
func detectPlatform() string {
	os := runtime.GOOS
	arch := runtime.GOARCH

	// Map Go architecture names to release asset names
	switch arch {
	case "amd64":
		arch = "amd64"
	case "386":
		arch = "386"
	case "arm64":
		arch = "arm64"
	case "arm":
		arch = "armv7"
	default:
		arch = "amd64" // fallback
	}

	return fmt.Sprintf("%s-%s", os, arch)
}

// downloadBinary downloads the specified version of zana for the current platform
func downloadBinary(version, platform string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	// Construct download URL
	fileName := fmt.Sprintf("zana-%s", platform)
	if platform == "windows-amd64" || platform == "windows-386" {
		fileName += ".exe"
	}

	downloadURL := fmt.Sprintf("https://github.com/mistweaverco/zana-client/releases/download/%s/%s", version, fileName)

	// Create temporary file
	tempFile, err := os.CreateTemp("", "zana-update-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer tempFile.Close()

	// Download the binary
	resp, err := client.Get(downloadURL)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to download binary: HTTP %d", resp.StatusCode)
	}

	// Copy the response to the temporary file
	_, err = io.Copy(tempFile, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("failed to save binary: %w", err)
	}

	// Make the file executable on Unix-like systems
	if platform != "windows-amd64" && platform != "windows-386" {
		if err := os.Chmod(tempFile.Name(), 0755); err != nil {
			os.Remove(tempFile.Name())
			return "", fmt.Errorf("failed to make binary executable: %w", err)
		}
	}

	return tempFile.Name(), nil
}

// backupCurrentBinary creates a backup of the current binary
func backupCurrentBinary(binaryPath string) (string, error) {
	timestamp := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.backup.%s", binaryPath, timestamp)

	if err := copyFile(binaryPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// replaceBinary replaces the current binary with the new one
func replaceBinary(currentPath, newBinaryPath string) error {
	// Remove the current binary
	if err := os.Remove(currentPath); err != nil {
		return fmt.Errorf("failed to remove current binary: %w", err)
	}

	// Copy the new binary to the current location
	if err := copyFile(newBinaryPath, currentPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	return nil
}

// runSelfUpdate executes the self-update process
func runSelfUpdate(output OutputWriter) error {
	// Get current version
	currentVersion := getCurrentVersion()
	if currentVersion == "" {
		return fmt.Errorf("current version is not set")
	}

	output.Printf("Current version: %s\n", currentVersion)

	// Get latest version
	output.Printf("Checking for updates...\n")
	latestVersion, err := getLatestVersion()
	if err != nil {
		return fmt.Errorf("failed to get latest version: %w", err)
	}

	// Compare versions
	if !semver.IsGreater(currentVersion, latestVersion) {
		output.Printf("zana is already up to date (version %s)\n", currentVersion)
		return nil
	}

	output.Printf("New version available: %s (current: %s)\n", latestVersion, currentVersion)

	// Get current binary path
	currentPath, err := getCurrentBinaryPath()
	if err != nil {
		return fmt.Errorf("failed to get current binary path: %w", err)
	}

	// Detect platform
	platform := detectPlatform()
	output.Printf("Detected platform: %s\n", platform)

	// Download new binary
	output.Printf("Downloading zana %s for %s...\n", latestVersion, platform)
	newBinaryPath, err := downloadBinary(latestVersion, platform)
	if err != nil {
		return fmt.Errorf("failed to download new version: %w", err)
	}
	defer os.Remove(newBinaryPath) // Clean up temp file

	// Create backup
	output.Printf("Creating backup of current binary...\n")
	backupPath, err := backupCurrentBinary(currentPath)
	if err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	output.Printf("Backup created: %s\n", backupPath)

	// Replace binary
	output.Printf("Installing new version...\n")
	if err := replaceBinary(currentPath, newBinaryPath); err != nil {
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	output.Printf("✓ Successfully updated zana from %s to %s\n", currentVersion, latestVersion)
	output.Printf("Backup saved as: %s\n", backupPath)

	return nil
}
