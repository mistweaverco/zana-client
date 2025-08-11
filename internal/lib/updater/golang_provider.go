package updater

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
)

type GolangProvider struct {
	APP_PACKAGES_DIR string
	PREFIX           string
}

func NewProviderGolang() *GolangProvider {
	p := &GolangProvider{}
	p.APP_PACKAGES_DIR = filepath.Join(files.GetAppPackagesPath(), "golang")
	p.PREFIX = "pkg:golang/"
	return p
}

func (p *GolangProvider) getRepo(sourceID string) string {
	re := regexp.MustCompile("^" + p.PREFIX + "(.*)")
	matches := re.FindStringSubmatch(sourceID)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (p *GolangProvider) generatePackageJSON() bool {
	found := false
	packageJSON := struct {
		Dependencies map[string]string `json:"dependencies"`
	}{
		Dependencies: make(map[string]string),
	}

	localPackages := local_packages_parser.GetData(true).Packages
	for _, pkg := range localPackages {
		if detectProvider(pkg.SourceID) != ProviderGolang {
			continue
		}
		packageJSON.Dependencies[p.getRepo(pkg.SourceID)] = pkg.Version
		found = true
	}

	filePath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating package.json:", err)
		return false
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close package.json file: %v\n", closeErr)
		}
	}()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(packageJSON)
	if err != nil {
		fmt.Println("Error encoding package.json:", err)
		return false
	}

	return found
}

// createSymlinks creates symlinks for Go binaries
func (p *GolangProvider) createSymlinks() error {
	golangBinDir := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	zanaBinDir := files.GetAppBinPath()

	// Check if golang bin directory exists
	if _, err := os.Stat(golangBinDir); os.IsNotExist(err) {
		return nil // No binaries installed
	}

	// Read all files in the golang bin directory
	entries, err := os.ReadDir(golangBinDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			binaryName := entry.Name()
			binaryPath := filepath.Join(golangBinDir, binaryName)
			symlinkPath := filepath.Join(zanaBinDir, binaryName)

			// Remove existing symlink if it exists
			if _, err := os.Lstat(symlinkPath); err == nil {
				if err := os.Remove(symlinkPath); err != nil {
					log.Printf("Warning: failed to remove existing symlink %s: %v", symlinkPath, err)
				}
			}

			// Create the symlink
			err := os.Symlink(binaryPath, symlinkPath)
			if err != nil {
				log.Printf("Error creating symlink for %s: %v", binaryName, err)
				continue
			}

			// Make the symlink executable
			err = os.Chmod(symlinkPath, 0755)
			if err != nil {
				log.Printf("Error setting executable permissions for %s: %v", binaryName, err)
			}
		}
	}

	return nil
}

// removeAllSymlinks removes all symlinks from the zana bin directory
func (p *GolangProvider) removeAllSymlinks() error {
	zanaBinDir := files.GetAppBinPath()

	// Read all files in the zana bin directory
	entries, err := os.ReadDir(zanaBinDir)
	if err != nil {
		return err
	}

	// Remove all symlinks
	for _, entry := range entries {
		if !entry.IsDir() {
			symlinkPath := filepath.Join(zanaBinDir, entry.Name())
			if _, err := os.Lstat(symlinkPath); err == nil {
				if err := os.Remove(symlinkPath); err != nil {
					log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
				}
			}
		}
	}

	return nil
}

// removePackageSymlinks removes symlinks for a specific package
func (p *GolangProvider) removePackageSymlinks(packageName string) error {
	zanaBinDir := files.GetAppBinPath()

	// For Go packages, the binary name is typically the base name of the package
	binaryName := filepath.Base(packageName)
	symlinkPath := filepath.Join(zanaBinDir, binaryName)

	if _, err := os.Lstat(symlinkPath); err == nil {
		log.Printf("Golang Remove: Removing symlink %s for package %s", binaryName, packageName)
		if err := os.Remove(symlinkPath); err != nil {
			log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
		}
	}

	// Also check for common variations of the binary name
	// Some Go packages might have different naming conventions
	commonNames := []string{
		binaryName,
		"go-" + binaryName,
		"golang-" + binaryName,
	}

	for _, name := range commonNames {
		symlinkPath := filepath.Join(zanaBinDir, name)
		if _, err := os.Lstat(symlinkPath); err == nil {
			log.Printf("Golang Remove: Removing symlink %s for package %s", name, packageName)
			if err := os.Remove(symlinkPath); err != nil {
				log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
			}
		}
	}

	return nil
}

func (p *GolangProvider) Clean() bool {
	// Remove all symlinks first
	err := p.removeAllSymlinks()
	if err != nil {
		log.Printf("Error removing symlinks: %v", err)
	}

	err = os.RemoveAll(p.APP_PACKAGES_DIR)
	if err != nil {
		log.Println("Error removing directory:", err)
		return false
	}
	return p.Sync()
}

// checkGoAvailable checks if the go command is available
func (p *GolangProvider) checkGoAvailable() bool {
	checkCode, err := shell_out.ShellOut("go", []string{"version"}, p.APP_PACKAGES_DIR, nil)
	return err == nil && checkCode == 0
}

func (p *GolangProvider) Sync() bool {
	if _, err := os.Stat(p.APP_PACKAGES_DIR); os.IsNotExist(err) {
		err := os.Mkdir(p.APP_PACKAGES_DIR, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return false
		}
	}

	// Check if Go is available
	if !p.checkGoAvailable() {
		log.Println("Error: Go is not available. Please install Go and ensure it's in your PATH.")
		return false
	}

	packagesFound := p.generatePackageJSON()
	if !packagesFound {
		return true
	}

	log.Printf("Golang Sync: Starting sync process")

	// Get desired packages from local_packages_parser
	desired := local_packages_parser.GetData(true).Packages

	// Initialize Go module if it doesn't exist
	goModPath := filepath.Join(p.APP_PACKAGES_DIR, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		initCode, err := shell_out.ShellOut("go", []string{"mod", "init", "zana-golang-packages"}, p.APP_PACKAGES_DIR, nil)
		if err != nil || initCode != 0 {
			log.Println("Error initializing Go module:", err)
			return false
		}
	}

	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	allOk := true
	installedCount := 0
	skippedCount := 0

	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)
		binPath := filepath.Join(gobin, filepath.Base(name))
		installed := false
		if fi, err := os.Stat(binPath); err == nil && !fi.IsDir() {
			// Optionally, check version by running the binary with --version if supported
			// For now, assume installed if binary exists
			installed = true
		}
		if !installed {
			log.Printf("Golang Sync: Installing package %s@%s", name, pkg.Version)
			installCode, err := shell_out.ShellOut("go", []string{"install", name + "@" + pkg.Version}, p.APP_PACKAGES_DIR, []string{"GOBIN=" + gobin})
			if err != nil || installCode != 0 {
				log.Printf("Error installing %s@%s: %v", name, pkg.Version, err)
				allOk = false
			} else {
				installedCount++
				err = p.createSymlinks()
				if err != nil {
					log.Printf("Error creating symlinks for %s: %v", name, err)
				}
			}
		} else {
			log.Printf("Golang Sync: Package %s@%s already installed, skipping", name, pkg.Version)
			skippedCount++
		}
	}

	log.Printf("Golang Sync: Completed - %d packages installed, %d packages skipped", installedCount, skippedCount)

	return allOk
}

func (p *GolangProvider) Install(sourceID, version string) bool {
	err := local_packages_parser.AddLocalPackage(sourceID, version)
	if err != nil {
		return false
	}
	return p.Sync()
}

func (p *GolangProvider) Remove(sourceID string) bool {
	// Get the package name before removing it from local packages
	packageName := p.getRepo(sourceID)

	log.Printf("Golang Remove: Removing package %s", packageName)

	// Remove symlinks for this package first
	err := p.removePackageSymlinks(packageName)
	if err != nil {
		log.Printf("Error removing symlinks for %s: %v", packageName, err)
		// Don't fail the remove if symlink removal fails
	}

	err = local_packages_parser.RemoveLocalPackage(sourceID)
	if err != nil {
		log.Printf("Error removing package %s from local packages: %v", packageName, err)
		return false
	}

	log.Printf("Golang Remove: Package %s removed successfully", packageName)
	return p.Sync()
}

func (p *GolangProvider) Update(sourceID string) bool {
	repo := p.getRepo(sourceID)
	if repo == "" {
		log.Printf("Invalid source ID format for Golang provider")
		return false
	}

	// Get the latest version from Go modules
	latestVersion, err := p.getLatestVersion(repo)
	if err != nil {
		log.Printf("Error getting latest version for %s: %v", repo, err)
		return false
	}

	log.Printf("Golang Update: Updating %s to version %s", repo, latestVersion)

	// Install the latest version
	return p.Install(sourceID, latestVersion)
}

func (p *GolangProvider) getLatestVersion(packageName string) (string, error) {
	// Use go list to get the latest version
	_, output, err := shell_out.ShellOutCapture("go", []string{"list", "-m", "-versions", packageName}, "", nil)
	if err != nil {
		return "", err
	}

	// Parse the output to get the latest version
	// Output format: "module version1 version2 version3 ..."
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid output format from go list")
	}

	// Return the last version (latest)
	return parts[len(parts)-1], nil
}
