package local_packages_parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

// marshalIndent is a package-level variable to allow injection during tests
var marshalIndent = json.MarshalIndent

type LocalPackageItem struct {
	SourceID string `json:"sourceId"`
	Version  string `json:"version"`
}

type LocalPackageRoot struct {
	Packages []LocalPackageItem `json:"packages"`
}

// LocalPackagesParser implements LocalPackagesManager
type LocalPackagesParser struct {
	fileManager FileManager
}

// New creates a new LocalPackagesParser with the default file manager
func New() *LocalPackagesParser {
	return &LocalPackagesParser{
		fileManager: &DefaultFileManager{},
	}
}

// NewWithFileManager creates a new LocalPackagesParser with a custom file manager
func NewWithFileManager(fileManager FileManager) *LocalPackagesParser {
	return &LocalPackagesParser{
		fileManager: fileManager,
	}
}

// normalizePackageID converts a package ID from legacy format (pkg:provider/pkg)
// to the new format (provider:pkg), or returns it unchanged if already in new format.
// This ensures backward compatibility when reading zana-lock.json files.
func normalizePackageID(sourceID string) string {
	if strings.HasPrefix(sourceID, "pkg:") {
		rest := strings.TrimPrefix(sourceID, "pkg:")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 2 {
			return parts[0] + ":" + parts[1]
		}
	}
	return sourceID
}

// GetData returns the local packages data from the local packages file.
// The force flag is ignored; data is always read from disk to avoid caching.
// Package IDs are normalized from legacy format (pkg:provider/pkg) to new format (provider:pkg)
// for backward compatibility.
func (lpp *LocalPackagesParser) GetData(force bool) LocalPackageRoot {
	localPackagesFile := lpp.fileManager.GetAppLocalPackagesFilePath()
	var localPackageRoot LocalPackageRoot

	if !lpp.fileManager.FileExists(localPackagesFile) {
		return LocalPackageRoot{Packages: []LocalPackageItem{}}
	}

	byteValue, err := lpp.fileManager.ReadFile(localPackagesFile)
	if err != nil {
		fmt.Printf("Warning: failed to read local packages file: %v\n", err)
		return LocalPackageRoot{Packages: []LocalPackageItem{}}
	}

	if err := json.Unmarshal(byteValue, &localPackageRoot); err != nil {
		fmt.Printf("Warning: failed to parse local packages file: %v\n", err)
		return LocalPackageRoot{Packages: []LocalPackageItem{}}
	}

	// Normalize all package IDs from legacy format to new format
	for i := range localPackageRoot.Packages {
		localPackageRoot.Packages[i].SourceID = normalizePackageID(localPackageRoot.Packages[i].SourceID)
	}

	return localPackageRoot
}

// GetDataForProvider returns the local packages data
// for a specific provider. Supports both legacy (pkg:provider/pkg) and new (provider:pkg) formats.
func (lpp *LocalPackagesParser) GetDataForProvider(provider string) LocalPackageRoot {
	localPackageRoot := lpp.GetData(false)
	filteredPackages := []LocalPackageItem{}

	for _, item := range localPackageRoot.Packages {
		// Check for new format: provider:pkg
		if strings.HasPrefix(item.SourceID, provider+":") {
			filteredPackages = append(filteredPackages, item)
		}
		// Also check legacy format for backward compatibility (though GetData normalizes)
		if strings.HasPrefix(item.SourceID, "pkg:"+provider+"/") {
			filteredPackages = append(filteredPackages, item)
		}
	}

	return LocalPackageRoot{Packages: filteredPackages}
}

func (lpp *LocalPackagesParser) AddLocalPackage(sourceId string, version string) error {
	// Normalize the source ID to new format before storing
	normalizedID := normalizePackageID(sourceId)
	localPackageRoot := lpp.GetData(false)
	packageExists := false

	// Check if the package is already installed (compare normalized IDs)
	for i, pkg := range localPackageRoot.Packages {
		if pkg.SourceID == normalizedID {
			// Update the existing package with the new version
			localPackageRoot.Packages[i].Version = version
			packageExists = true
			break
		}
	}

	// If not found, add the new package with normalized ID
	if !packageExists {
		localPackageRoot.Packages = append(localPackageRoot.Packages, LocalPackageItem{
			SourceID: normalizedID,
			Version:  version,
		})
	}

	localPackagesFile := lpp.fileManager.GetAppLocalPackagesFilePath()
	jsonData, err := marshalIndent(localPackageRoot, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	if err := lpp.fileManager.WriteFile(localPackagesFile, jsonData, 0644); err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	return nil
}

func (lpp *LocalPackagesParser) RemoveLocalPackage(sourceId string) error {
	// Normalize the source ID to new format before looking up
	normalizedID := normalizePackageID(sourceId)
	localPackageRoot := lpp.GetData(false)
	for i, pkg := range localPackageRoot.Packages {
		if pkg.SourceID == normalizedID {
			localPackageRoot.Packages = append(localPackageRoot.Packages[:i], localPackageRoot.Packages[i+1:]...)
			break
		}
	}

	localPackagesFile := lpp.fileManager.GetAppLocalPackagesFilePath()
	jsonData, err := marshalIndent(localPackageRoot, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	if err := lpp.fileManager.WriteFile(localPackagesFile, jsonData, 0644); err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	return nil
}

func (lpp *LocalPackagesParser) GetBySourceId(sourceId string) LocalPackageItem {
	// Normalize the source ID to new format before looking up
	normalizedID := normalizePackageID(sourceId)
	localPackageRoot := lpp.GetData(false)
	for _, item := range localPackageRoot.Packages {
		if item.SourceID == normalizedID {
			return item
		}
	}
	return LocalPackageItem{}
}

func (lpp *LocalPackagesParser) IsPackageInstalled(sourceId string) bool {
	// Normalize the source ID to new format before looking up
	normalizedID := normalizePackageID(sourceId)
	localPackageRoot := lpp.GetData(false)
	for _, item := range localPackageRoot.Packages {
		if item.SourceID == normalizedID {
			return true
		}
	}
	return false
}

// Global instance for backward compatibility
var globalParser *LocalPackagesParser

func init() {
	globalParser = New()
}

// Legacy functions for backward compatibility
func GetData(force bool) LocalPackageRoot {
	return globalParser.GetData(force)
}

func GetDataForProvider(provider string) LocalPackageRoot {
	return globalParser.GetDataForProvider(provider)
}

func AddLocalPackage(sourceId string, version string) error {
	return globalParser.AddLocalPackage(sourceId, version)
}

func RemoveLocalPackage(sourceId string) error {
	return globalParser.RemoveLocalPackage(sourceId)
}

func GetBySourceId(sourceId string) LocalPackageItem {
	return globalParser.GetBySourceId(sourceId)
}

func IsPackageInstalled(sourceId string) bool {
	return globalParser.IsPackageInstalled(sourceId)
}
