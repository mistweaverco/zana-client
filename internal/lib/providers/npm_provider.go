package providers

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

type PackageJSON struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Bin     map[string]string `json:"bin"`
}

func (p *NPMProvider) readPackageJSON(packagePath string) (*PackageJSON, error) {
	packageJSONPath := filepath.Join(packagePath, "package.json")
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, err
	}
	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

func (p *NPMProvider) createSymlinks(packagePath string, pkg *PackageJSON) error {
	binDir := files.GetAppBinPath()
	for binName, binPath := range pkg.Bin {
		actualBinPath := filepath.Join(packagePath, binPath)
		symlinkPath := filepath.Join(binDir, binName)
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				log.Printf("Warning: failed to remove existing symlink %s: %v", symlinkPath, err)
			}
		}
		if err := os.Symlink(actualBinPath, symlinkPath); err != nil {
			log.Printf("Error creating symlink for %s: %v", binName, err)
			return err
		}
		if err := os.Chmod(symlinkPath, 0755); err != nil {
			log.Printf("Error setting executable permissions for %s: %v", binName, err)
		}
	}
	return nil
}

func (p *NPMProvider) removeAllSymlinks() error {
	binDir := files.GetAppBinPath()
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		symlinkPath := filepath.Join(binDir, entry.Name())
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
			}
		}
	}
	return nil
}

func (p *NPMProvider) Clean() bool {
	if err := p.removeAllSymlinks(); err != nil {
		log.Printf("Error removing symlinks: %v", err)
	}
	if err := os.RemoveAll(p.APP_PACKAGES_DIR); err != nil {
		log.Println("Error removing directory:", err)
		return false
	}
	return p.Sync()
}

func (p *NPMProvider) Sync() bool {
	if _, err := os.Stat(p.APP_PACKAGES_DIR); os.IsNotExist(err) {
		if err := os.Mkdir(p.APP_PACKAGES_DIR, 0755); err != nil {
			fmt.Println("Error creating directory:", err)
			return false
		}
	}
	log.Printf("NPM Sync: Starting sync process")
	packagesFound := p.generatePackageJSON()
	if !packagesFound {
		return true
	}
	desired := local_packages_parser.GetDataForProvider("npm").Packages
	lockFile := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	packageJSONFile := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	lockExists := false
	lockNewer := false
	if lockStat, err := os.Stat(lockFile); err == nil {
		lockExists = true
		if pkgStat, err := os.Stat(packageJSONFile); err == nil {
			lockNewer = lockStat.ModTime().After(pkgStat.ModTime())
		}
	}
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
		if allInstalled {
			log.Printf("NPM Sync: All packages already installed correctly, skipping installation")
			for _, pkg := range desired {
				name := p.getRepo(pkg.SourceID)
				if err := p.createPackageSymlinks(name); err != nil {
					log.Printf("Error creating symlinks for %s: %v", name, err)
				}
			}
			return true
		}
	}
	if lockExists && lockNewer {
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
		if allInstalled {
			for _, pkg := range desired {
				name := p.getRepo(pkg.SourceID)
				if err := p.createPackageSymlinks(name); err != nil {
					log.Printf("Error creating symlinks for %s: %v", name, err)
				}
			}
			return true
		}
		if needsUpdate {
			log.Printf("NPM Sync: Attempting npm ci for faster bulk installation")
			if p.tryNpmCi() {
				log.Printf("NPM Sync: npm ci completed successfully")
				return true
			}
			log.Printf("NPM Sync: npm ci failed, falling back to individual package installation")
			if err := os.Remove(lockFile); err != nil {
				log.Printf("Warning: failed to remove lock file: %v", err)
			}
		}
	}
	log.Printf("NPM Sync: Installing packages individually")
	allOk := true
	installedCount := 0
	skippedCount := 0
	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)
		if p.isPackageInstalled(name, pkg.Version) {
			log.Printf("NPM Sync: Package %s@%s already installed, skipping", name, pkg.Version)
			skippedCount++
			if err := p.createPackageSymlinks(name); err != nil {
				log.Printf("Error creating symlinks for %s: %v", name, err)
			}
			continue
		}
		log.Printf("NPM Sync: Installing package %s@%s", name, pkg.Version)
		installCode, err := shell_out.ShellOut("npm", []string{"install", name + "@" + pkg.Version}, p.APP_PACKAGES_DIR, nil)
		if err != nil || installCode != 0 {
			fmt.Printf("Error installing %s@%s: %v\n", name, pkg.Version, err)
			allOk = false
		} else {
			installedCount++
			if err := p.createPackageSymlinks(name); err != nil {
				log.Printf("Error creating symlinks for %s: %v", name, err)
			}
		}
	}
	log.Printf("NPM Sync: Completed - %d packages installed, %d packages skipped", installedCount, skippedCount)
	return allOk
}

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

func (p *NPMProvider) isPackageInstalled(packageName, expectedVersion string) bool {
	packagePath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", packageName)
	if _, err := os.Stat(packagePath); os.IsNotExist(err) {
		return false
	}
	pkg, err := p.readPackageJSON(packagePath)
	if err != nil {
		return false
	}
	return pkg.Version == expectedVersion
}

func (p *NPMProvider) createPackageSymlinks(packageName string) error {
	nodeModulesPath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules")
	packagePath := filepath.Join(nodeModulesPath, packageName)
	pkg, err := p.readPackageJSON(packagePath)
	if err != nil {
		return fmt.Errorf("error reading package.json for %s: %v", packageName, err)
	}
	if len(pkg.Bin) > 0 {
		if err := p.createSymlinks(packagePath, pkg); err != nil {
			return fmt.Errorf("error creating symlinks for %s: %v", packageName, err)
		}
	}
	return nil
}

func (p *NPMProvider) removePackageSymlinks(packageName string) error {
	binDir := files.GetAppBinPath()
	nodeModulesPath := filepath.Join(p.APP_PACKAGES_DIR, "node_modules")
	packagePath := filepath.Join(nodeModulesPath, packageName)
	pkg, err := p.readPackageJSON(packagePath)
	if err != nil {
		return nil
	}
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
	packageName := p.getRepo(sourceID)
	if err := local_packages_parser.AddLocalPackage(sourceID, version); err != nil {
		return false
	}
	success := p.Sync()
	if success {
		if err := p.createPackageSymlinks(packageName); err != nil {
			log.Printf("Error creating symlinks for %s: %v", packageName, err)
		}
	}
	return success
}

func (p *NPMProvider) Remove(sourceID string) bool {
	packageName := p.getRepo(sourceID)
	log.Printf("NPM Remove: Removing package %s", packageName)
	if err := p.removePackageSymlinks(packageName); err != nil {
		log.Printf("Error removing symlinks for %s: %v", packageName, err)
	}
	if err := local_packages_parser.RemoveLocalPackage(sourceID); err != nil {
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
	latestVersion, err := p.getLatestVersion(repo)
	if err != nil {
		log.Printf("Error getting latest version for %s: %v", repo, err)
		return false
	}
	log.Printf("NPM Update: Updating %s to version %s", repo, latestVersion)
	return p.Install(sourceID, latestVersion)
}

func (p *NPMProvider) getLatestVersion(packageName string) (string, error) {
	_, output, err := shell_out.ShellOutCapture("npm", []string{"view", packageName, "version"}, "", nil)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (p *NPMProvider) tryNpmCi() bool {
	lockFile := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		log.Printf("NPM Sync: No package-lock.json found, cannot use npm ci")
		return false
	}
	log.Printf("NPM Sync: Using npm ci for faster bulk installation")
	installCode, err := shell_out.ShellOut("npm", []string{"ci"}, p.APP_PACKAGES_DIR, nil)
	if err != nil || installCode != 0 {
		log.Printf("NPM Sync: npm ci failed, falling back to individual package installation: %v", err)
		return false
	}
	log.Printf("NPM Sync: npm ci completed successfully, creating symlinks")
	desired := local_packages_parser.GetData(true).Packages
	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)
		if err := p.createPackageSymlinks(name); err != nil {
			log.Printf("Error creating symlinks for %s: %v", name, err)
		}
	}
	return true
}

func (p *NPMProvider) hasPackageJSONChanged() bool {
	packageJSONFile := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	lockFile := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	if _, err := os.Stat(packageJSONFile); os.IsNotExist(err) {
		return true
	}
	if _, err := os.Stat(lockFile); os.IsNotExist(err) {
		return true
	}
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
