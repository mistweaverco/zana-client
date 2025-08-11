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

type NPMProvider struct {
	APP_PACKAGES_DIR string
	PREFIX           string
	PROVIDER_NAME    string
}

func NewProviderNPM() *NPMProvider {
	p := &NPMProvider{}
	p.PROVIDER_NAME = "npm"
	p.APP_PACKAGES_DIR = filepath.Join(files.GetAppPackagesPath(), p.PROVIDER_NAME)
	p.PREFIX = "pkg:" + p.PROVIDER_NAME + "/"
	return p
}

func (p *NPMProvider) getRepo(sourceID string) string {
	re := regexp.MustCompile("^" + p.PREFIX + "(.*)")
	matches := re.FindStringSubmatch(sourceID)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (p *NPMProvider) generatePackageJSON() bool {
	found := false
	packageJSON := struct {
		Dependencies map[string]string `json:"dependencies"`
	}{
		Dependencies: make(map[string]string),
	}

	localPackages := local_packages_parser.GetData(true).Packages
	for _, pkg := range localPackages {
		if detectProvider(pkg.SourceID) != ProviderNPM {
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

// PackageJSON represents the structure of an npm package.json file
type PackageJSON struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Bin     map[string]string `json:"bin"`
}

// readPackageJSON reads and parses a package.json file
func (p *NPMProvider) readPackageJSON(packagePath string) (*PackageJSON, error) {
	packageJSONPath := filepath.Join(packagePath, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	err = json.Unmarshal(data, &pkg)
	if err != nil {
		return nil, err
	}

	return &pkg, nil
}

// createSymlinks creates symlinks for the bin entries in a package
func (p *NPMProvider) createSymlinks(packagePath string, pkg *PackageJSON) error {
	binDir := files.GetAppBinPath()

	for binName, binPath := range pkg.Bin {
		// Resolve the actual path to the binary within the package
		actualBinPath := filepath.Join(packagePath, binPath)

		// Create the symlink path in the bin directory
		symlinkPath := filepath.Join(binDir, binName)

		// Remove existing symlink if it exists
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				log.Printf("Warning: failed to remove existing symlink %s: %v", symlinkPath, err)
			}
		}

		// Create the symlink
		err := os.Symlink(actualBinPath, symlinkPath)
		if err != nil {
			log.Printf("Error creating symlink for %s: %v", binName, err)
			return err
		}

		// Make the symlink executable
		err = os.Chmod(symlinkPath, 0755)
		if err != nil {
			log.Printf("Error setting executable permissions for %s: %v", binName, err)
		}
	}

	return nil
}

// removeAllSymlinks removes all symlinks from the bin directory
func (p *NPMProvider) removeAllSymlinks() error {
	binDir := files.GetAppBinPath()

	// Read all files in the bin directory
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}

	// Remove all symlinks
	for _, entry := range entries {
		if !entry.IsDir() {
			symlinkPath := filepath.Join(binDir, entry.Name())
			if _, err := os.Lstat(symlinkPath); err == nil {
				if err := os.Remove(symlinkPath); err != nil {
					log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
				}
			}
		}
	}

	return nil
}

func (p *NPMProvider) Clean() bool {
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

func (p *NPMProvider) Sync() bool {
	if _, err := os.Stat(p.APP_PACKAGES_DIR); os.IsNotExist(err) {
		err := os.Mkdir(p.APP_PACKAGES_DIR, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return false
		}
	}

	log.Printf("NPM Sync: Starting sync process")

	packagesFound := p.generatePackageJSON()
	if !packagesFound {
		return true
	}

	// Get desired packages from local_packages_parser
	desired := local_packages_parser.GetDataForProvider("npm").Packages

	// Check if we have a package-lock.json and if it's up to date
	lockFile := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	packageJSONFile := filepath.Join(p.APP_PACKAGES_DIR, "package.json")

	// Check if package-lock.json exists and is newer than package.json
	lockExists := false
	lockNewer := false
	if lockStat, err := os.Stat(lockFile); err == nil {
		lockExists = true
		if pkgStat, err := os.Stat(packageJSONFile); err == nil {
			lockNewer = lockStat.ModTime().After(pkgStat.ModTime())
		}
	}

	// Early exit: if package.json hasn't changed and we have a valid lock file,
	// check if all packages are already installed correctly
	if lockExists && lockNewer && !p.hasPackageJSONChanged() {
		installed := p.getInstalledPackagesFromLock(lockFile)
		allInstalled := true

		for _, pkg := range desired {
			name := p.getRepo(pkg.SourceID)
			if v, ok := installed[name]; !ok || v != pkg.Version {
				allInstalled = false
				break
			}
		}

		// If all packages are already installed correctly, just create symlinks and exit
		if allInstalled {
			log.Printf("NPM Sync: All packages already installed correctly, skipping installation")
			for _, pkg := range desired {
				name := p.getRepo(pkg.SourceID)
				err := p.createPackageSymlinks(name)
				if err != nil {
					log.Printf("Error creating symlinks for %s: %v", name, err)
				}
			}
			return true
		}
	}

	// If we have a valid lock file, try to use npm ci for faster installation
	if lockExists && lockNewer {
		// Check if all desired packages are already installed with correct versions
		installed := p.getInstalledPackagesFromLock(lockFile)
		allInstalled := true
		needsUpdate := false

		for _, pkg := range desired {
			name := p.getRepo(pkg.SourceID)
			if v, ok := installed[name]; !ok || v != pkg.Version {
				allInstalled = false
				needsUpdate = true
				break
			}
		}

		// If all packages are already installed correctly, just create symlinks
		if allInstalled {
			for _, pkg := range desired {
				name := p.getRepo(pkg.SourceID)
				err := p.createPackageSymlinks(name)
				if err != nil {
					log.Printf("Error creating symlinks for %s: %v", name, err)
				}
			}
			return true
		}

		// If we need updates but have a lock file, try npm ci first for faster installation
		if needsUpdate {
			log.Printf("NPM Sync: Attempting npm ci for faster bulk installation")
			// Try to use npm ci for faster bulk installation
			if p.tryNpmCi() {
				log.Printf("NPM Sync: npm ci completed successfully")
				return true
			}
			log.Printf("NPM Sync: npm ci failed, falling back to individual package installation")
			// If npm ci fails, remove the lock file to force a fresh install
			if err := os.Remove(lockFile); err != nil {
				log.Printf("Warning: failed to remove lock file: %v", err)
			}
		}
	}

	// Fall back to individual package installation
	log.Printf("NPM Sync: Installing packages individually")
	allOk := true
	installedCount := 0
	skippedCount := 0

	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)

		// Check if package is already installed with correct version
		if p.isPackageInstalled(name, pkg.Version) {
			// Package is already installed, just create symlinks
			log.Printf("NPM Sync: Package %s@%s already installed, skipping", name, pkg.Version)
			skippedCount++
			err := p.createPackageSymlinks(name)
			if err != nil {
				log.Printf("Error creating symlinks for %s: %v", name, err)
			}
			continue
		}

		// Install package if not installed or version mismatch
		log.Printf("NPM Sync: Installing package %s@%s", name, pkg.Version)
		installCode, err := shell_out.ShellOut("npm", []string{"install", name + "@" + pkg.Version}, p.APP_PACKAGES_DIR, nil)
		if err != nil || installCode != 0 {
			fmt.Printf("Error installing %s@%s: %v\n", name, pkg.Version, err)
			allOk = false
		} else {
			installedCount++
			err = p.createPackageSymlinks(name)
			if err != nil {
				log.Printf("Error creating symlinks for %s: %v", name, err)
			}
		}
	}

	log.Printf("NPM Sync: Completed - %d packages installed, %d packages skipped", installedCount, skippedCount)

	return allOk
}

// getInstalledPackagesFromLock reads package-lock.json and returns a map of installed packages
func (p *NPMProvider) getInstalledPackagesFromLock(lockFile string) map[string]string {
	installed := map[string]string{}

	data, err := os.ReadFile(lockFile)
	if err != nil {
		return installed
	}

	var lock struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	if err := json.Unmarshal(data, &lock); err == nil {
		for pkg, info := range lock.Dependencies {
			installed[pkg] = info.Version
		}
	}

	return installed
}

// isPackageInstalled checks if a specific package is installed with the correct version
func (p *NPMProvider) isPackageInstalled(packageName, expectedVersion string) bool {
	// Check if package directory exists
	packagePath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", packageName)
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return false
	}

	// Read package.json to check version
	pkg, err := p.readPackageJSON(packagePath)
	if err != nil {
		return false
	}

	return pkg.Version == expectedVersion
}

// createPackageSymlinks creates symlinks for a specific package
func (p *NPMProvider) createPackageSymlinks(packageName string) error {
	nodeModulesPath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules")
	packagePath := filepath.Join(nodeModulesPath, packageName)

	pkg, err := p.readPackageJSON(packagePath)
	if err != nil {
		return fmt.Errorf("error reading package.json for %s: %v", packageName, err)
	}

	// Create symlinks if the package has bin entries
	if len(pkg.Bin) > 0 {
		err = p.createSymlinks(packagePath, pkg)
		if err != nil {
			return fmt.Errorf("error creating symlinks for %s: %v", packageName, err)
		}
	}

	return nil
}

// removePackageSymlinks removes symlinks for a specific package
func (p *NPMProvider) removePackageSymlinks(packageName string) error {
	binDir := files.GetAppBinPath()

	// Read the package.json to get bin entries
	nodeModulesPath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules")
	packagePath := filepath.Join(nodeModulesPath, packageName)
	pkg, err := p.readPackageJSON(packagePath)
	if err != nil {
		// Package might not exist anymore, which is fine
		return nil
	}

	// Remove symlinks for each bin entry
	for binName := range pkg.Bin {
		symlinkPath := filepath.Join(binDir, binName)
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
			}
		}
	}

	return nil
}

func (p *NPMProvider) Install(sourceID, version string) bool {
	// Get the package name before adding it to local packages
	packageName := p.getRepo(sourceID)

	err := local_packages_parser.AddLocalPackage(sourceID, version)
	if err != nil {
		return false
	}

	// Sync to install the package
	success := p.Sync()

	// If sync was successful, create symlinks for this specific package
	if success {
		err = p.createPackageSymlinks(packageName)
		if err != nil {
			log.Printf("Error creating symlinks for %s: %v", packageName, err)
			// Don't fail the install if symlink creation fails
		}
	}

	return success
}

func (p *NPMProvider) Remove(sourceID string) bool {
	// Get the package name before removing it from local packages
	packageName := p.getRepo(sourceID)

	log.Printf("NPM Remove: Removing package %s", packageName)

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

	log.Printf("NPM Remove: Package %s removed successfully", packageName)
	return p.Sync()
}

func (p *NPMProvider) Update(sourceID string) bool {
	repo := p.getRepo(sourceID)
	if repo == "" {
		log.Printf("Invalid source ID format for NPM provider")
		return false
	}

	// Get the latest version from npm
	latestVersion, err := p.getLatestVersion(repo)
	if err != nil {
		log.Printf("Error getting latest version for %s: %v", repo, err)
		return false
	}

	log.Printf("NPM Update: Updating %s to version %s", repo, latestVersion)

	// Install the latest version
	return p.Install(sourceID, latestVersion)
}

func (p *NPMProvider) getLatestVersion(packageName string) (string, error) {
	// Use npm view to get the latest version
	_, output, err := shell_out.ShellOutCapture("npm", []string{"view", packageName, "version"}, "", nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// tryNpmCi attempts to install all packages using npm ci for better performance
func (p *NPMProvider) tryNpmCi() bool {
	// Check if we have a valid package-lock.json
	lockFile := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		log.Printf("NPM Sync: No package-lock.json found, cannot use npm ci")
		return false
	}

	log.Printf("NPM Sync: Using npm ci for faster bulk installation")
	// Try to use npm ci for faster installation
	installCode, err := shell_out.ShellOut("npm", []string{"ci"}, p.APP_PACKAGES_DIR, nil)
	if err != nil || installCode != 0 {
		log.Printf("NPM Sync: npm ci failed, falling back to individual package installation: %v", err)
		return false
	}

	log.Printf("NPM Sync: npm ci completed successfully, creating symlinks")
	// npm ci succeeded, create symlinks for all packages
	desired := local_packages_parser.GetData(true).Packages
	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)
		err := p.createPackageSymlinks(name)
		if err != nil {
			log.Printf("Error creating symlinks for %s: %v", name, err)
		}
	}

	return true
}

// hasPackageJSONChanged checks if package.json has been modified since the last sync
func (p *NPMProvider) hasPackageJSONChanged() bool {
	packageJSONFile := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	lockFile := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")

	// If package.json doesn't exist, consider it changed
	if _, err := os.Stat(packageJSONFile); os.IsNotExist(err) {
		return true
	}

	// If lock file doesn't exist, consider it changed
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		return true
	}

	// Check if package.json is newer than package-lock.json
	pkgStat, err := os.Stat(packageJSONFile)
	if err != nil {
		return true
	}

	lockStat, err := os.Stat(lockFile)
	if err != nil {
		return true
	}

	return pkgStat.ModTime().After(lockStat.ModTime())
}
