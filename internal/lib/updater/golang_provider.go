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
				os.Remove(symlinkPath)
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
				os.Remove(symlinkPath)
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

	returnResult := true

	filePath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	packageJSON, err := os.ReadFile(filePath)
	if err != nil {
		log.Println("Error reading golang package.json:", err)
		return false
	}

	pkgJSON := struct {
		Dependencies map[string]string `json:"dependencies"`
	}{
		Dependencies: make(map[string]string),
	}

	err = json.Unmarshal(packageJSON, &pkgJSON)
	if err != nil {
		log.Println("Error unmarshalling golang package.json:", err)
		return false
	}

	// Initialize Go module if it doesn't exist
	goModPath := filepath.Join(p.APP_PACKAGES_DIR, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		initCode, err := shell_out.ShellOut("go", []string{
			"mod", "init", "zana-golang-packages",
		}, p.APP_PACKAGES_DIR, nil)
		if err != nil || initCode != 0 {
			log.Println("Error initializing Go module:", err)
			return false
		}
	}

	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")

	for pkg, version := range pkgJSON.Dependencies {
		// Try different package paths for the main executable
		packagePaths := []string{
			pkg,                                // Try the package as-is
			pkg + "/" + filepath.Base(pkg),     // Try package/package (common pattern)
			pkg + "/cmd/" + filepath.Base(pkg), // Try package/cmd/package
		}

		installed := false
		for _, packagePath := range packagePaths {
			// Try go install with version
			installCode, err := shell_out.ShellOut("go", []string{
				"install",
				packagePath + "@" + version,
			}, p.APP_PACKAGES_DIR,
				[]string{
					"GOBIN=" + gobin,
				})

			// If that fails, try without version (latest)
			if err != nil || installCode != 0 {
				log.Printf("Failed to install %s@%s, trying latest version", packagePath, version)
				installCode, err = shell_out.ShellOut("go", []string{
					"install",
					packagePath,
				}, p.APP_PACKAGES_DIR,
					[]string{
						"GOBIN=" + gobin,
					})
			}

			if err == nil && installCode == 0 {
				log.Printf("Successfully installed %s", packagePath)
				installed = true
				break
			}
		}

		if !installed {
			log.Printf("Failed to install package: %s", pkg)
			returnResult = false
		}
	}

	// Create symlinks for installed binaries
	if returnResult {
		err = p.createSymlinks()
		if err != nil {
			log.Printf("Error creating symlinks: %v", err)
			// Don't fail the sync if symlink creation fails
		}
	}

	return returnResult
}

func (p *GolangProvider) Install(sourceID, version string) bool {
       packageName := p.getRepo(sourceID)
       gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
       binaryName := filepath.Base(packageName)
       binaryPath := filepath.Join(gobin, binaryName)

       // Check if binary exists
       if _, err := os.Stat(binaryPath); err == nil {
	       // Optionally, check version using 'go version -m' (Go 1.18+)
	       // If not possible, just skip install if binary exists
	       return true
       }

       err := local_packages_parser.AddLocalPackage(sourceID, version)
       if err != nil {
	       return false
       }
       return p.Sync()
}

func (p *GolangProvider) Remove(sourceID string) bool {
	err := local_packages_parser.RemoveLocalPackage(sourceID)
	if err != nil {
		return false
	}
	return p.Sync()
}
