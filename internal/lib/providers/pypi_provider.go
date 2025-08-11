package providers

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
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
			_ = fmt.Errorf("warning: failed to close requirements.txt file: %v", closeErr)
		}
	}()
	return found
}

// PackageInfo represents the structure of a Python package's metadata
type PackageInfo struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	EntryPoints map[string]interface{} `json:"entry_points,omitempty"`
}

// readPackageInfo reads package metadata from installed Python packages
func (p *PyPiProvider) readPackageInfo(packagePath string) (*PackageInfo, error) {
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return nil, err
	}

	var infoDir string
	for _, entry := range entries {
		if entry.IsDir() && (strings.HasSuffix(entry.Name(), ".dist-info") || strings.HasSuffix(entry.Name(), ".egg-info")) {
			infoDir = filepath.Join(packagePath, entry.Name())
			break
		}
	}
	if infoDir == "" {
		return nil, fmt.Errorf("no package info directory found")
	}

	metadataFiles := []string{"METADATA", "PKG-INFO"}
	var metadataContent string
	for _, filename := range metadataFiles {
		metadataPath := filepath.Join(infoDir, filename)
		if data, err := os.ReadFile(metadataPath); err == nil {
			metadataContent = string(data)
			break
		}
	}
	if metadataContent == "" {
		return nil, fmt.Errorf("no metadata file found")
	}

	lines := strings.Split(metadataContent, "\n")
	info := &PackageInfo{}
	for _, line := range lines {
		if strings.HasPrefix(line, "Name: ") {
			info.Name = strings.TrimSpace(strings.TrimPrefix(line, "Name: "))
		} else if strings.HasPrefix(line, "Version: ") {
			info.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version: "))
		}
	}
	return info, nil
}

// createWrappers creates wrapper scripts for Python package scripts
func (p *PyPiProvider) createWrappers() error {
	// Create wrappers based on zana-registry.json bin attribute
	desired := local_packages_parser.GetDataForProvider("pypi").Packages
	if len(desired) == 0 {
		return nil
	}
	zanaBinDir := files.GetAppBinPath()
	for _, pkg := range desired {
		registryItem := registry_parser.GetBySourceId(pkg.SourceID)
		if len(registryItem.Bin) == 0 {
			continue
		}
		for binName, binCmd := range registryItem.Bin {
			wrapperPath := filepath.Join(zanaBinDir, binName)
			// Remove any existing wrapper with the same name to avoid conflicts
			if _, err := os.Lstat(wrapperPath); err == nil {
				_ = os.Remove(wrapperPath)
			}
			if err := p.createPythonWrapperForCommand(binCmd, wrapperPath); err != nil {
				Logger.Error(fmt.Sprintf("Error creating wrapper for %s: %v", binName, err))
				continue
			}
			if err := os.Chmod(wrapperPath, 0755); err != nil {
				Logger.Error(fmt.Sprintf("Error setting executable permissions for %s: %v", binName, err))
			}
		}
	}
	return nil
}

// createPythonWrapperForCommand creates a wrapper that prepares the environment and executes the given command.
func (p *PyPiProvider) createPythonWrapperForCommand(commandToExec string, wrapperPath string) error {
	sitePackagesDir := p.findSitePackagesDir()
	binDir := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	if sitePackagesDir == "" {
		sitePackagesDir = p.APP_PACKAGES_DIR
	}
	if commandToExec == "" {
		return fmt.Errorf("empty command for wrapper %s", wrapperPath)
	}
	wrapperContent := fmt.Sprintf(`#!/bin/sh
# Sets up Python environment for zana-installed packages and runs the target command

# Add the zana Python packages to PYTHONPATH and PATH (to resolve console scripts)
export PYTHONPATH="%s:$PYTHONPATH"
export PATH="%s:$PATH"

# Execute the command from registry
exec %s "$@"
`, sitePackagesDir, binDir, commandToExec)
	if err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755); err != nil {
		return err
	}
	return nil
}

// findSitePackagesDir finds the site-packages directory where pip installed the modules
func (p *PyPiProvider) findSitePackagesDir() string {
	libDir := filepath.Join(p.APP_PACKAGES_DIR, "lib")
	if _, err := os.Stat(libDir); os.IsNotExist(err) {
		return ""
	}
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "python") {
			sitePackagesPath := filepath.Join(libDir, entry.Name(), "site-packages")
			if _, err := os.Stat(sitePackagesPath); err == nil {
				return sitePackagesPath
			}
		}
	}
	return ""
}

// removeAllWrappers removes all wrapper scripts from the zana bin directory
func (p *PyPiProvider) removeAllWrappers() error {
	// Remove only wrappers managed by the PyPI provider based on registry bin names
	desired := local_packages_parser.GetDataForProvider("pypi").Packages
	if len(desired) == 0 {
		return nil
	}
	zanaBinDir := files.GetAppBinPath()
	for _, pkg := range desired {
		registryItem := registry_parser.GetBySourceId(pkg.SourceID)
		for binName := range registryItem.Bin {
			wrapperPath := filepath.Join(zanaBinDir, binName)
			if _, err := os.Lstat(wrapperPath); err == nil {
				if err := os.Remove(wrapperPath); err != nil {
					Logger.Error(fmt.Sprintf("Warning: failed to remove wrapper script %s: %v", wrapperPath, err))
				}
			}
		}
	}
	return nil
}

// removePackageWrappers removes wrapper scripts for a specific package
func (p *PyPiProvider) removePackageWrappers(packageName string) error {
	zanaBinDir := files.GetAppBinPath()
	// Reconstruct sourceId to query registry
	sourceID := p.PREFIX + packageName
	registryItem := registry_parser.GetBySourceId(sourceID)
	if len(registryItem.Bin) == 0 {
		return nil
	}
	for binName := range registryItem.Bin {
		wrapperPath := filepath.Join(zanaBinDir, binName)
		if _, err := os.Lstat(wrapperPath); err == nil {
			if err := os.Remove(wrapperPath); err != nil {
				Logger.Error(fmt.Sprintf("Warning: failed to remove wrapper script %s: %v", wrapperPath, err))
			}
		}
	}
	return nil
}

// normalizeDistributionName normalizes a distribution name per PEP 503
func normalizeDistributionName(name string) string {
	n := strings.ToLower(name)
	n = strings.ReplaceAll(n, "_", "-")
	return n
}

// findPackageInfoDir tries to locate the <dist>-<version>.dist-info (or .egg-info) directory for a package
func (p *PyPiProvider) findPackageInfoDir(packageName string) string {
	sitePackagesDir := p.findSitePackagesDir()
	if sitePackagesDir == "" {
		return ""
	}
	normalized := normalizeDistributionName(packageName)
	entries, err := os.ReadDir(sitePackagesDir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !(strings.HasSuffix(name, ".dist-info") || strings.HasSuffix(name, ".egg-info")) {
			continue
		}
		// Trim suffix and compare the leading portion with normalized package name
		base := strings.TrimSuffix(strings.TrimSuffix(name, ".dist-info"), ".egg-info")
		baseNorm := normalizeDistributionName(base)
		if strings.HasPrefix(baseNorm, normalized+"-") || baseNorm == normalized {
			return filepath.Join(sitePackagesDir, name)
		}
	}
	return ""
}

// parseEntryPointsFromInfoDir parses entry_points.txt to get executable names (console_scripts/gui_scripts)
func (p *PyPiProvider) parseEntryPointsFromInfoDir(infoDir string) []string {
	epPath := filepath.Join(infoDir, "entry_points.txt")
	data, err := os.ReadFile(epPath)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var result []string
	currentSection := ""
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			currentSection = strings.ToLower(strings.Trim(line, "[]"))
			continue
		}
		if currentSection == "console_scripts" || currentSection == "gui_scripts" {
			if idx := strings.Index(line, "="); idx != -1 {
				name := strings.TrimSpace(line[:idx])
				if name != "" {
					result = append(result, name)
				}
			}
		}
	}
	return result
}

func (p *PyPiProvider) Clean() bool {
	if err := p.removeAllWrappers(); err != nil {
		Logger.Error(fmt.Sprintf("Error removing wrapper scripts: %v", err))
	}
	if err := os.RemoveAll(p.APP_PACKAGES_DIR); err != nil {
		Logger.Error("Error removing directory:")
		return false
	}
	return p.Sync()
}

func (p *PyPiProvider) Sync() bool {
	if _, err := os.Stat(p.APP_PACKAGES_DIR); os.IsNotExist(err) {
		if err := os.Mkdir(p.APP_PACKAGES_DIR, 0755); err != nil {
			fmt.Println("Error creating directory:", err)
			return false
		}
	}

	packagesFound := p.generateRequirementsTxt()
	if !packagesFound {
		return true
	}

	Logger.Info("PyPI Sync: Starting sync process")

	desired := local_packages_parser.GetDataForProvider("pypi").Packages

	if p.areAllPackagesInstalled(desired) {
		Logger.Info("PyPI Sync: All packages already installed correctly, skipping installation")
		if err := p.createWrappers(); err != nil {
			Logger.Error(fmt.Sprintf("Error creating wrapper scripts: %v", err))
		}
		return true
	}

	installed := p.getInstalledPackages()
	allOk := true
	installedCount := 0
	skippedCount := 0

	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)
		if v, ok := installed[name]; !ok || v != pkg.Version {
			pkgString := fmt.Sprintf("%s==%s", name, pkg.Version)
			Logger.Info(fmt.Sprintf("PyPI Sync: Installing package %s", pkgString))
			installCode, err := shell_out.ShellOut(pipCmd, []string{"install", pkgString, "--prefix", p.APP_PACKAGES_DIR}, p.APP_PACKAGES_DIR, nil)
			if err != nil || installCode != 0 {
				Logger.Error(fmt.Sprintf("Error installing %s==%s: %v", name, pkg.Version, err))
				allOk = false
			} else {
				installedCount++
			}
		} else {
			Logger.Info(fmt.Sprintf("PyPI Sync: Package %s==%s already installed, skipping", name, pkg.Version))
			skippedCount++
		}
	}

	if allOk {
		if err := p.createWrappers(); err != nil {
			Logger.Error(fmt.Sprintf("Error creating wrapper scripts: %v", err))
		}
	}

	Logger.Info(fmt.Sprintf("PyPI Sync: Completed - %d packages installed, %d packages skipped", installedCount, skippedCount))
	return allOk
}

// areAllPackagesInstalled checks if all desired packages are already installed with correct versions
func (p *PyPiProvider) areAllPackagesInstalled(desired []local_packages_parser.LocalPackageItem) bool {
	installed := p.getInstalledPackages()
	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)
		if v, ok := installed[name]; !ok || v != pkg.Version {
			return false
		}
	}
	return true
}

// getInstalledPackages gets the list of installed packages using pip freeze
func (p *PyPiProvider) getInstalledPackages() map[string]string {
	installed := map[string]string{}
	freezeCode, freezeOut, _ := shell_out.ShellOutCapture(pipCmd, []string{"freeze"}, p.APP_PACKAGES_DIR, nil)
	if freezeCode == 0 && freezeOut != "" {
		lines := strings.Split(freezeOut, "\n")
		for _, line := range lines {
			parts := strings.Split(line, "==")
			if len(parts) == 2 {
				installed[parts[0]] = parts[1]
			}
		}
	} else {
		Logger.Error(fmt.Sprintf("Error getting installed packages with %s freeze: %s", pipCmd, freezeOut))
	}
	return installed
}

func (p *PyPiProvider) Install(sourceID, version string) bool {
	var err error
	if version == "latest" {
		version, err = p.getLatestVersion(p.getRepo(sourceID))
		if err != nil {
			Logger.Error(fmt.Sprintf("Error getting latest version for %s: %v", sourceID, err))
			return false
		}
	}
	if err = local_packages_parser.AddLocalPackage(sourceID, version); err != nil {
		Logger.Error(fmt.Sprintf("Error adding package %s to local packages: %v", sourceID, err))
		return false
	}
	return p.Sync()
}

func (p *PyPiProvider) removeBin(sourceID string) error {
	packageName := p.getRepo(sourceID)
	if packageName == "" {
		return fmt.Errorf("invalid source ID format for PyPI provider")
	}
	Logger.Info(fmt.Sprintf("PyPI Remove: Removing bin for package %s", packageName))
	infoDir := p.findPackageInfoDir(packageName)
	if infoDir == "" {
		return fmt.Errorf("no package info directory found for %s", packageName)
	}
	entryPoints := p.parseEntryPointsFromInfoDir(infoDir)
	for _, entryPoint := range entryPoints {
		binPath := filepath.Join(p.APP_PACKAGES_DIR, "bin", entryPoint)
		if _, err := os.Lstat(binPath); err == nil {
			if err := os.Remove(binPath); err != nil {
				return fmt.Errorf("failed to remove bin script %s: %v", binPath, err)
			}
		}
	}
	return nil
}

func (p *PyPiProvider) Remove(sourceID string) bool {
	packageName := p.getRepo(sourceID)
	Logger.Info(fmt.Sprintf("PyPI Remove: Removing package %s", packageName))
	if err := p.removeBin(sourceID); err != nil {
		Logger.Error(fmt.Sprintf("Error removing bin for package %s: %v", packageName, err))
		return false
	}
	if err := p.removePackageWrappers(packageName); err != nil {
		Logger.Error(fmt.Sprintf("Error removing wrapper scripts for %s: %v", packageName, err))
	}
	if err := local_packages_parser.RemoveLocalPackage(sourceID); err != nil {
		Logger.Error(fmt.Sprintf("Error removing package %s from local packages: %v", packageName, err))
		return false
	}
	Logger.Info(fmt.Sprintf("PyPI Remove: Package %s removed successfully", packageName))
	return p.Sync()
}

func (p *PyPiProvider) Update(sourceID string) bool {
	repo := p.getRepo(sourceID)
	if repo == "" {
		Logger.Error("Invalid source ID format for PyPI provider")
		return false
	}
	latestVersion, err := p.getLatestVersion(repo)
	if err != nil {
		Logger.Error(fmt.Sprintf("Error getting latest version for %s: %v", repo, err))
		return false
	}
	Logger.Info(fmt.Sprintf("PyPI Update: Updating %s to version %s", repo, latestVersion))
	return p.Install(sourceID, latestVersion)
}

func (p *PyPiProvider) getLatestVersion(packageName string) (string, error) {
	_, output, err := shell_out.ShellOutCapture(pipCmd, []string{"index", "versions", packageName}, "", nil)
	if err != nil {
		return "", err
	}
	start := strings.LastIndex(output, "(")
	end := strings.LastIndex(output, ")")
	if start == -1 || end == -1 {
		return "", fmt.Errorf("invalid output format from pip index")
	}
	versionsStr := output[start+1 : end]
	versions := strings.Split(versionsStr, ", ")
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}
	return strings.TrimSpace(versions[len(versions)-1]), nil
}
