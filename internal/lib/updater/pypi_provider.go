package updater

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
)

type PyPiProvider struct {
	APP_PACKAGES_DIR string
	PREFIX           string
}

func NewProviderPyPi() *PyPiProvider {
	p := &PyPiProvider{}
	p.APP_PACKAGES_DIR = filepath.Join(files.GetAppPackagesPath(), "pypi")
	p.PREFIX = "pkg:pypi/"
	return p
}

func (p *PyPiProvider) getRepo(sourceID string) string {
	re := regexp.MustCompile("^" + p.PREFIX + "(.*)")
	matches := re.FindStringSubmatch(sourceID)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (p *PyPiProvider) generateRequirementsTxt() bool {
	found := false
	dependenciesTxt := make([]string, 0)

	localPackages := local_packages_parser.GetData(true).Packages
	for _, pkg := range localPackages {
		if detectProvider(pkg.SourceID) != ProviderPyPi {
			continue
		}
		dependenciesTxt = append(dependenciesTxt, fmt.Sprintf("%s==%s", p.getRepo(pkg.SourceID), pkg.Version))
		found = true
	}

	filePath := filepath.Join(p.APP_PACKAGES_DIR, "requirements.txt")
	file, err := os.Create(filePath)
	if err != nil {
		log.Println("Error creating requirements.txt:", err)
		return false
	}

	for _, line := range dependenciesTxt {
		_, err := file.WriteString(line + "\n")
		if err != nil {
			log.Println("Error writing to requirements.txt:", err)
			return false
		}
	}

	defer file.Close()

	return found
}

func (p *PyPiProvider) Clean() bool {
	err := os.RemoveAll(p.APP_PACKAGES_DIR)
	if err != nil {
		log.Println("Error removing directory:", err)
		return false
	}
	return p.syncPackages()
}

func (p *PyPiProvider) syncPackages() bool {
	if _, err := os.Stat(p.APP_PACKAGES_DIR); os.IsNotExist(err) {
		err := os.Mkdir(p.APP_PACKAGES_DIR, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			return false
		}
	}

	packagesFound := p.generateRequirementsTxt()

	if !packagesFound {
		return true
	}

	// TODO: find a non-hacky way to prune the packages in python

	installCode, err := shell_out.ShellOut("pip", []string{
		"install",
		"-r",
		"requirements.txt",
		"--target",
		"pkgs",
	}, p.APP_PACKAGES_DIR, nil)
	if err != nil || installCode != 0 {
		log.Println("Error running pip install:", err)
		return false
	}

	return installCode == 0
}

func (p *PyPiProvider) Install(sourceID, version string) bool {
	err := local_packages_parser.AddLocalPackage(sourceID, version)
	if err != nil {
		return false
	}
	return p.syncPackages()
}

func (p *PyPiProvider) Remove(sourceID string) bool {
	err := local_packages_parser.RemoveLocalPackage(sourceID)
	if err != nil {
		return false
	}
	return p.syncPackages()
}
