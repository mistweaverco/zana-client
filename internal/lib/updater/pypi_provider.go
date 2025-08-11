package updater

import (
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

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close requirements.txt file: %v\n", closeErr)
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
	// Try to find the package info in the installed package
	// Look for .dist-info or .egg-info directories
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

	// Try to read METADATA or PKG-INFO file
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

	// Parse basic metadata (simplified parsing)
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
	zanaBinDir := files.GetAppBinPath()

	// Check if packages directory exists
	if _, err := os.Stat(p.APP_PACKAGES_DIR); os.IsNotExist(err) {
		return nil // No packages installed
	}

	// With --prefix, pip installs scripts to bin directory
	binDir := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	if _, err := os.Stat(binDir); os.IsNotExist(err) {
		return nil // No bin directory
	}

	// Read all files in the bin directory
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}

	// First, remove all existing wrapper scripts to ensure clean state
	err = p.removeAllWrappers()
	if err != nil {
		log.Printf("Error removing existing wrapper scripts: %v", err)
	}

	// Create wrapper scripts for all executable files in the bin directory
	for _, entry := range entries {
		if !entry.IsDir() {
			scriptName := entry.Name()
			scriptPath := filepath.Join(binDir, scriptName)
			wrapperPath := filepath.Join(zanaBinDir, scriptName)

			// Create a wrapper script that sets up the Python path
			err := p.createPythonWrapper(scriptPath, wrapperPath)
			if err != nil {
				log.Printf("Error creating wrapper for %s: %v", scriptName, err)
				continue
			}

			// Make the wrapper executable
			err = os.Chmod(wrapperPath, 0755)
			if err != nil {
				log.Printf("Error setting executable permissions for %s: %v", scriptName, err)
			}
		}
	}

	return nil
}

// createPythonWrapper creates a wrapper script that sets up the Python path
func (p *PyPiProvider) createPythonWrapper(originalScript, wrapperPath string) error {
	// Find the site-packages directory
	sitePackagesDir := p.findSitePackagesDir()
	if sitePackagesDir == "" {
		// Fallback to the main packages directory
		sitePackagesDir = p.APP_PACKAGES_DIR
	}

	// Create the wrapper script content
	wrapperContent := fmt.Sprintf(`#!/bin/sh
# Wrapper script for %s
# Sets up Python path to include zana-installed packages

# Add the zana Python packages to PYTHONPATH
export PYTHONPATH="%s:$PYTHONPATH"

# Execute the original script
exec "%s" "$@"
`, filepath.Base(originalScript), sitePackagesDir, originalScript)

	// Write the wrapper script
	err := os.WriteFile(wrapperPath, []byte(wrapperContent), 0755)
	if err != nil {
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

	// Look for python* directories in lib
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
	zanaBinDir := files.GetAppBinPath()

	// Read all files in the zana bin directory
	entries, err := os.ReadDir(zanaBinDir)
	if err != nil {
		return err
	}

	// Remove all wrapper scripts
	for _, entry := range entries {
		if !entry.IsDir() {
			wrapperPath := filepath.Join(zanaBinDir, entry.Name())
					if _, err := os.Lstat(wrapperPath); err == nil {
			if err := os.Remove(wrapperPath); err != nil {
				log.Printf("Warning: failed to remove wrapper script %s: %v", wrapperPath, err)
			}
		}
		}
	}

	return nil
}

// removePackageWrappers removes wrapper scripts for a specific package
func (p *PyPiProvider) removePackageWrappers(packageName string) error {
	zanaBinDir := files.GetAppBinPath()

	// Get the package info to see what entry points it has
	// We need to check the site-packages directory where the package was installed
	sitePackagesDir := p.findSitePackagesDir()
	if sitePackagesDir == "" {
		// Fallback to the main packages directory
		sitePackagesDir = p.APP_PACKAGES_DIR
	}

	packagePath := filepath.Join(sitePackagesDir, packageName)
	pkgInfo, err := p.readPackageInfo(packagePath)
	if err != nil {
		// Package might not exist anymore, which is fine
		return nil
	}

	// Remove wrapper scripts for each entry point
	if pkgInfo.EntryPoints != nil {
		for entryPoint := range pkgInfo.EntryPoints {
			wrapperPath := filepath.Join(zanaBinDir, entryPoint)
			if _, err := os.Lstat(wrapperPath); err == nil {
				log.Printf("PyPI Remove: Removing wrapper script %s for package %s", entryPoint, packageName)
				if err := os.Remove(wrapperPath); err != nil {
					log.Printf("Warning: failed to remove wrapper script %s: %v", wrapperPath, err)
				}
			}
		}
	}

	// Also check for common script names that might have been created
	// This is a fallback for packages that don't have explicit entry points
	commonScriptNames := []string{packageName, "python-" + packageName}
	for _, scriptName := range commonScriptNames {
		wrapperPath := filepath.Join(zanaBinDir, scriptName)
		if _, err := os.Lstat(wrapperPath); err == nil {
			log.Printf("PyPI Remove: Removing wrapper script %s for package %s", scriptName, packageName)
			if err := os.Remove(wrapperPath); err != nil {
				log.Printf("Warning: failed to remove wrapper script %s: %v", wrapperPath, err)
			}
		}
	}

	return nil
}

func (p *PyPiProvider) Clean() bool {
	// Remove all wrapper scripts first
	err := p.removeAllWrappers()
	if err != nil {
		log.Printf("Error removing wrapper scripts: %v", err)
	}

	err = os.RemoveAll(p.APP_PACKAGES_DIR)
	if err != nil {
		log.Println("Error removing directory:", err)
		return false
	}
	return p.Sync()
}

func (p *PyPiProvider) Sync() bool {
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

	log.Printf("PyPI Sync: Starting sync process")

	// Get desired packages from local_packages_parser
	desired := local_packages_parser.GetData(true).Packages

	// Check if we have a requirements.txt and if it's up to date
	_ = filepath.Join(p.APP_PACKAGES_DIR, "requirements.txt")

	// Early exit: check if all packages are already installed correctly
	if p.areAllPackagesInstalled(desired) {
		log.Printf("PyPI Sync: All packages already installed correctly, skipping installation")
		err := p.createWrappers()
		if err != nil {
			log.Printf("Error creating wrapper scripts: %v", err)
		}
		return true
	}

	// Get installed packages using pip freeze
	installed := p.getInstalledPackages()

	allOk := true
	installedCount := 0
	skippedCount := 0

	for _, pkg := range desired {
		name := p.getRepo(pkg.SourceID)
		if v, ok := installed[name]; !ok || v != pkg.Version {
			log.Printf("PyPI Sync: Installing package %s==%s", name, pkg.Version)
			installCode, err := shell_out.ShellOut("pip", []string{"install", name + "==" + pkg.Version, "--prefix", p.APP_PACKAGES_DIR}, p.APP_PACKAGES_DIR, nil)
			if err != nil || installCode != 0 {
				log.Printf("Error installing %s==%s: %v", name, pkg.Version, err)
				allOk = false
			} else {
				installedCount++
			}
		} else {
			log.Printf("PyPI Sync: Package %s==%s already installed, skipping", name, pkg.Version)
			skippedCount++
		}
	}

	// Create wrappers for all packages
	if allOk {
		err := p.createWrappers()
		if err != nil {
			log.Printf("Error creating wrapper scripts: %v", err)
		}
	}

	log.Printf("PyPI Sync: Completed - %d packages installed, %d packages skipped", installedCount, skippedCount)

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

	pipCmd := "pip3"
	freezeCode, freezeOut, _ := shell_out.ShellOutCapture(pipCmd, []string{"freeze"}, p.APP_PACKAGES_DIR, nil)
	if freezeCode != 0 || freezeOut == "" {
		pipCmd = "pip"
		freezeCode, freezeOut, _ = shell_out.ShellOutCapture(pipCmd, []string{"freeze"}, p.APP_PACKAGES_DIR, nil)
	}

	if freezeCode == 0 && freezeOut != "" {
		lines := strings.Split(freezeOut, "\n")
		for _, line := range lines {
			parts := strings.Split(line, "==")
			if len(parts) == 2 {
				installed[parts[0]] = parts[1]
			}
		}
	}

	return installed
}

func (p *PyPiProvider) Install(sourceID, version string) bool {
	err := local_packages_parser.AddLocalPackage(sourceID, version)
	if err != nil {
		return false
	}
	return p.Sync()
}

func (p *PyPiProvider) Remove(sourceID string) bool {
	// Get the package name before removing it from local packages
	packageName := p.getRepo(sourceID)

	log.Printf("PyPI Remove: Removing package %s", packageName)

	// Remove wrapper scripts for this package first
	err := p.removePackageWrappers(packageName)
	if err != nil {
		log.Printf("Error removing wrapper scripts for %s: %v", packageName, err)
		// Don't fail the remove if wrapper removal fails
	}

	err = local_packages_parser.RemoveLocalPackage(sourceID)
	if err != nil {
		log.Printf("Error removing package %s from local packages: %v", packageName, err)
		return false
	}

	log.Printf("PyPI Remove: Package %s removed successfully", packageName)
	return p.Sync()
}
