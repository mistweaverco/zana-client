package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
)

type NPMProvider struct{}

// Assuming you have Go equivalents for these functions/types
// Replace with your actual implementations
type PackageInfo struct {
	SourceID string `json:"sourceId"`
	Version  string `json:"version"`
}

var APP_PACKAGES_NPM_DIR = filepath.Join(files.GetAppPackagesPath(), "npm")

func getRepo(sourceID string) string {
	re := regexp.MustCompile(`^pkg:npm/(.*)`)
	matches := re.FindStringSubmatch(sourceID)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func generatePackageJSON() bool {
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
		packageJSON.Dependencies[getRepo(pkg.SourceID)] = pkg.Version
		found = true
	}

	filePath := filepath.Join(APP_PACKAGES_NPM_DIR, "package.json")
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

func syncPackages() bool {
	if _, err := os.Stat(APP_PACKAGES_NPM_DIR); os.IsNotExist(err) {
		err := os.Mkdir(APP_PACKAGES_NPM_DIR, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return false
		}
	}

	packagesFound := generatePackageJSON()

	if !packagesFound {
		return true
	}
	pruneCode, err := shell_out.ShellOut("npm", []string{"prune"}, APP_PACKAGES_NPM_DIR, nil)
	if err != nil || pruneCode != 0 {
		fmt.Println("Error running npm prune:", err)
		return false
	}

	installCode, err := shell_out.ShellOut("npm", []string{"install"}, APP_PACKAGES_NPM_DIR, nil)
	if err != nil || installCode != 0 {
		fmt.Println("Error running npm install:", err)
		return false
	}

	return installCode == 0
}

func (p *NPMProvider) Install(sourceID, version string) bool {
	err := local_packages_parser.AddLocalPackage(sourceID, version)
	if err != nil {
		return false
	}
	return syncPackages()
}

func (p *NPMProvider) Remove(sourceID string) bool {
	err := local_packages_parser.RemoveLocalPackage(sourceID)
	if err != nil {
		return false
	}
	return syncPackages()
}
