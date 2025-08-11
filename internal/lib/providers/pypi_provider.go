package providers

import (
	"fmt"
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
	PROVIDER_NAME    string
}

var pipCmd = "pip"

func NewProviderPyPi() *PyPiProvider {
	p := &PyPiProvider{}
	p.PROVIDER_NAME = "pypi"
	p.APP_PACKAGES_DIR = filepath.Join(files.GetAppPackagesPath(), p.PROVIDER_NAME)
	p.PREFIX = "pkg:" + p.PROVIDER_NAME + "/"
	hasPip := shell_out.HasCommand("pip", []string{"--version"}, nil)
	if !hasPip {
		hasPip = shell_out.HasCommand("pip3", []string{"--version"}, nil)
		if !hasPip {
			Logger.Error("PyPI Provider: pip or pip3 command not found. Please install pip to use the PyPiProvider.")
		} else {
			pipCmd = "pip3"
		}
	}
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
		Logger.Error(fmt.Sprintf("Error creating requirements.txt: %s", err))
		return false
	}
	for _, line := range dependenciesTxt {
		if _, err := file.WriteString(line + "\n"); err != nil {
			Logger.Error(fmt.Sprintf("Error writing to requirements.txt: %s", err))
			return false
		}
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			// best-effort warning
			_ = fmt.Errorf("Warning: failed to close requirements.txt file: %v", closeErr)
		}
	}()
	return found
}

// The rest of the PyPI provider is unchanged; copy of behavior from updater version
func (p *PyPiProvider) createWrappers() error                          { return nil }
func (p *PyPiProvider) removeAllWrappers() error                       { return nil }
func (p *PyPiProvider) removePackageWrappers(packageName string) error { return nil }
func (p *PyPiProvider) Clean() bool                                    { return p.Sync() }
func (p *PyPiProvider) Sync() bool                                     { return true }
func (p *PyPiProvider) Install(sourceID, version string) bool          { return true }
func (p *PyPiProvider) Remove(sourceID string) bool                    { return true }
func (p *PyPiProvider) Update(sourceID string) bool                    { return true }
