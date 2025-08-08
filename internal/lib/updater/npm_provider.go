package updater

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
)

type NPMProvider struct {
	APP_PACKAGES_DIR string
	PREFIX           string
}

func NewProviderNPM() *NPMProvider {
	p := &NPMProvider{}
	p.APP_PACKAGES_DIR = filepath.Join(files.GetAppPackagesPath(), "npm")
	p.PREFIX = "pkg:npm/"
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
	defer file.Close()

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
func (p *NPMProvider) createSymlinks(packageName string, packagePath string, pkg *PackageJSON) error {
	binDir := files.GetAppBinPath()

	for binName, binPath := range pkg.Bin {
		// Resolve the actual path to the binary within the package
		actualBinPath := filepath.Join(packagePath, binPath)

		// Create the symlink path in the bin directory
		symlinkPath := filepath.Join(binDir, binName)

		// Remove existing symlink if it exists
		if _, err := os.Lstat(symlinkPath); err == nil {
			os.Remove(symlinkPath)
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

// createAllSymlinks creates symlinks for all installed npm packages
func (p *NPMProvider) createAllSymlinks() error {
	// First, remove all existing symlinks to ensure clean state
	err := p.removeAllSymlinks()
	if err != nil {
		log.Printf("Error removing existing symlinks: %v", err)
	}

	nodeModulesPath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules")

	// Check if node_modules directory exists
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		return nil // No packages installed
	}

	// Get the list of packages managed by zana
	localPackages := local_packages_parser.GetData(true).Packages
	managedPackages := make(map[string]bool)

	for _, pkg := range localPackages {
		if detectProvider(pkg.SourceID) == ProviderNPM {
			packageName := p.getRepo(pkg.SourceID)
			managedPackages[packageName] = true
		}
	}

	// Only create symlinks for packages managed by zana
	for packageName := range managedPackages {
		packagePath := filepath.Join(nodeModulesPath, packageName)
		pkg, err := p.readPackageJSON(packagePath)
		if err != nil {
			log.Printf("Error reading package.json for %s: %v", packageName, err)
			continue
		}

		// Create symlinks if the package has bin entries
		if len(pkg.Bin) > 0 {
			err = p.createSymlinks(packageName, packagePath, pkg)
			if err != nil {
				log.Printf("Error creating symlinks for %s: %v", packageName, err)
			}
		}
	}

	return nil
}

// removeSymlinks removes symlinks for a specific package
func (p *NPMProvider) removeSymlinks(packageName string) error {
	binDir := files.GetAppBinPath()

	// Read the package.json to get bin entries
	packagePath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", packageName)
	pkg, err := p.readPackageJSON(packagePath)
	if err != nil {
		// Package might not exist anymore, which is fine
		return nil
	}

	// Remove symlinks for each bin entry
	for binName := range pkg.Bin {
		symlinkPath := filepath.Join(binDir, binName)
		if _, err := os.Lstat(symlinkPath); err == nil {
			os.Remove(symlinkPath)
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
				os.Remove(symlinkPath)
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

	packagesFound := p.generatePackageJSON()

	if !packagesFound {
		return true
	}
	pruneCode, err := shell_out.ShellOut("npm", []string{"prune"}, p.APP_PACKAGES_DIR, nil)
	if err != nil || pruneCode != 0 {
		fmt.Println("Error running npm prune:", err)
		return false
	}

	installCode, err := shell_out.ShellOut("npm", []string{"install"}, p.APP_PACKAGES_DIR, nil)
	if err != nil || installCode != 0 {
		fmt.Println("Error running npm install:", err)
		return false
	}

	// Recreate symlinks for all managed packages after sync
	if installCode == 0 {
		err = p.createAllSymlinks()
		if err != nil {
			log.Printf("Error creating symlinks: %v", err)
			// Don't fail the sync if symlink creation fails
		}
	}

	return installCode == 0
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
		err = p.createSymlinks(packageName, packagePath, pkg)
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
			os.Remove(symlinkPath)
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

	// Remove symlinks for this package first
	err := p.removePackageSymlinks(packageName)
	if err != nil {
		log.Printf("Error removing symlinks for %s: %v", packageName, err)
		// Don't fail the remove if symlink removal fails
	}

	err = local_packages_parser.RemoveLocalPackage(sourceID)
	if err != nil {
		return false
	}
	return p.Sync()
}
