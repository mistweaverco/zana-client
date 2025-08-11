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

type GolangProvider struct {
	APP_PACKAGES_DIR string
	PREFIX           string
	PROVIDER_NAME    string
}

func NewProviderGolang() *GolangProvider {
	p := &GolangProvider{}
	p.PROVIDER_NAME = "golang"
	p.APP_PACKAGES_DIR = filepath.Join(files.GetAppPackagesPath(), p.PROVIDER_NAME)
	p.PREFIX = "pkg:" + p.PROVIDER_NAME + "/"
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
	localPackages := local_packages_parser.GetDataForProvider("golang").Packages
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
	if err := encoder.Encode(packageJSON); err != nil {
		fmt.Println("Error encoding package.json:", err)
		return false
	}
	return found
}

func (p *GolangProvider) createSymlinks() error {
	golangBinDir := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	zanaBinDir := files.GetAppBinPath()
	if _, err := os.Stat(golangBinDir); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(golangBinDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		binaryName := entry.Name()
		binaryPath := filepath.Join(golangBinDir, binaryName)
		symlinkPath := filepath.Join(zanaBinDir, binaryName)
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				log.Printf("Warning: failed to remove existing symlink %s: %v", symlinkPath, err)
			}
		}
		if err := os.Symlink(binaryPath, symlinkPath); err != nil {
			log.Printf("Error creating symlink for %s: %v", binaryName, err)
			continue
		}
		if err := os.Chmod(symlinkPath, 0755); err != nil {
			log.Printf("Error setting executable permissions for %s: %v", binaryName, err)
		}
	}
	return nil
}

func (p *GolangProvider) removeAllSymlinks() error {
	zanaBinDir := files.GetAppBinPath()
	entries, err := os.ReadDir(zanaBinDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		symlinkPath := filepath.Join(zanaBinDir, entry.Name())
		if _, err := os.Lstat(symlinkPath); err == nil {
			if err := os.Remove(symlinkPath); err != nil {
				log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
			}
		}
	}
	return nil
}

func (p *GolangProvider) removePackageSymlinks(packageName string) error {
	zanaBinDir := files.GetAppBinPath()
	binaryName := filepath.Base(packageName)
	symlinkPath := filepath.Join(zanaBinDir, binaryName)
	if _, err := os.Lstat(symlinkPath); err == nil {
		log.Printf("Golang Remove: Removing symlink %s for package %s", binaryName, packageName)
		if err := os.Remove(symlinkPath); err != nil {
			log.Printf("Warning: failed to remove symlink %s: %v", symlinkPath, err)
		}
	}
	for _, name := range []string{binaryName, "go-" + binaryName, "golang-" + binaryName} {
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
	if err := p.removeAllSymlinks(); err != nil {
		log.Printf("Error removing symlinks: %v", err)
	}
	if err := os.RemoveAll(p.APP_PACKAGES_DIR); err != nil {
		log.Println("Error removing directory:", err)
		return false
	}
	return p.Sync()
}

func (p *GolangProvider) checkGoAvailable() bool {
	checkCode, err := shell_out.ShellOut("go", []string{"version"}, p.APP_PACKAGES_DIR, nil)
	return err == nil && checkCode == 0
}

func (p *GolangProvider) Sync() bool {
	if _, err := os.Stat(p.APP_PACKAGES_DIR); os.IsNotExist(err) {
		if err := os.Mkdir(p.APP_PACKAGES_DIR, 0755); err != nil {
			fmt.Println("Error creating directory:", err)
			return false
		}
	}
	if !p.checkGoAvailable() {
		log.Println("Error: Go is not available. Please install Go and ensure it's in your PATH.")
		return false
	}
	packagesFound := p.generatePackageJSON()
	if !packagesFound {
		return true
	}
	log.Printf("Golang Sync: Starting sync process")
	desired := local_packages_parser.GetDataForProvider("golang").Packages
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
				if err := p.createSymlinks(); err != nil {
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
	var err error
	if version == "latest" {
		version, err = p.getLatestVersion(p.getRepo(sourceID))
		if err != nil {
			Logger.Error("Error getting latest version for package %s: %v", sourceID, err)
			return false
		}
	}
	if err := local_packages_parser.AddLocalPackage(sourceID, version); err != nil {
		return false
	}
	return p.Sync()
}

func (p *GolangProvider) Remove(sourceID string) bool {
	packageName := p.getRepo(sourceID)
	log.Printf("Golang Remove: Removing package %s", packageName)
	if err := p.removePackageSymlinks(packageName); err != nil {
		log.Printf("Error removing symlinks for %s: %v", packageName, err)
	}
	if err := local_packages_parser.RemoveLocalPackage(sourceID); err != nil {
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
	latestVersion, err := p.getLatestVersion(repo)
	if err != nil {
		log.Printf("Error getting latest version for %s: %v", repo, err)
		return false
	}
	log.Printf("Golang Update: Updating %s to version %s", repo, latestVersion)
	return p.Install(sourceID, latestVersion)
}

func (p *GolangProvider) getLatestVersion(packageName string) (string, error) {
	_, output, err := shell_out.ShellOutCapture("go", []string{"list", "-m", "-versions", packageName}, "", nil)
	if err != nil {
		return "", err
	}
	parts := strings.Fields(output)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid output format from go list")
	}
	return parts[len(parts)-1], nil
}
