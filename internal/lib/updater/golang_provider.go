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

func (p *GolangProvider) Clean() bool {
	err := os.RemoveAll(p.APP_PACKAGES_DIR)
	if err != nil {
		log.Println("Error removing directory:", err)
		return false
	}
	return p.Sync()
}

func (p *GolangProvider) Sync() bool {
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

	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")

	for pkg, version := range pkgJSON.Dependencies {
		installCode, err := shell_out.ShellOut("go", []string{
			"install",
			pkg + "@" + version,
		}, p.APP_PACKAGES_DIR,
			[]string{
				"GOBIN=" + gobin,
			})
		if err != nil || installCode != 0 {
			log.Println("Error running go install:", err)
			log.Println("Tried installing package:", pkg+"@"+version, "in:", gobin)
			returnResult = false
		}
	}

	return returnResult
}

func (p *GolangProvider) Install(sourceID, version string) bool {
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
