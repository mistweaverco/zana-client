package local_packages_parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/files"
)

type LocalPackageItem struct {
	SourceID string `json:"sourceId"`
	Version  string `json:"version"`
}

type LocalPackageRoot struct {
	Packages []LocalPackageItem `json:"packages"`
}

var data LocalPackageRoot
var hasData bool

// GetData returns the local packages data
// from the local packages file
// if force is true, it will force to read the file
// and update the data
func GetData(force bool) LocalPackageRoot {
	if hasData && !force {
		return data
	}
	localPackagesFile := files.GetAppLocalPackagesFilePath()
	var localPackageRoot LocalPackageRoot
	jsonFile, err := os.Open(localPackagesFile)
	if err != nil {
		data = LocalPackageRoot{
			Packages: []LocalPackageItem{},
		}
		return data
	}
	defer func() {
		if closeErr := jsonFile.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close local packages file: %v\n", closeErr)
		}
	}()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Printf("Warning: failed to read local packages file: %v\n", err)
		return LocalPackageRoot{Packages: []LocalPackageItem{}}
	}
	if err := json.Unmarshal(byteValue, &localPackageRoot); err != nil {
		fmt.Printf("Warning: failed to parse local packages file: %v\n", err)
		return LocalPackageRoot{Packages: []LocalPackageItem{}}
	}
	hasData = true
	data = localPackageRoot
	return data
}

// GetDataForProvider returns the local packages data
// for a specific provider
func GetDataForProvider(provider string) LocalPackageRoot {
	localPackageRoot := GetData(false)
	filteredPackages := []LocalPackageItem{}

	for _, item := range localPackageRoot.Packages {
		if strings.HasPrefix(item.SourceID, "pkg:"+provider+"/") {
			filteredPackages = append(filteredPackages, item)
		}
	}

	return LocalPackageRoot{Packages: filteredPackages}
}

func AddLocalPackage(sourceId string, version string) error {
	localPackageRoot := GetData(true)
	packageExists := false

	// Check if the package is already installed
	for i, pkg := range localPackageRoot.Packages {
		if pkg.SourceID == sourceId {
			// Update the existing package with the new version
			localPackageRoot.Packages[i].Version = version
			packageExists = true
			break
		}
	}

	// If not found, add the new package
	if !packageExists {
		localPackageRoot.Packages = append(localPackageRoot.Packages, LocalPackageItem{
			SourceID: sourceId,
			Version:  version,
		})
	}

	localPackagesFile := files.GetAppLocalPackagesFilePath()
	jsonData, err := json.MarshalIndent(localPackageRoot, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	if err := os.WriteFile(localPackagesFile, jsonData, 0644); err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	return nil
}

func RemoveLocalPackage(sourceId string) error {
	localPackageRoot := GetData(true)
	for i, pkg := range localPackageRoot.Packages {
		if pkg.SourceID == sourceId {
			localPackageRoot.Packages = append(localPackageRoot.Packages[:i], localPackageRoot.Packages[i+1:]...)
			break
		}
	}

	localPackagesFile := files.GetAppLocalPackagesFilePath()
	jsonData, err := json.Marshal(localPackageRoot)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	if err := os.WriteFile(localPackagesFile, jsonData, 0644); err != nil {
		fmt.Println("Error writing to file:", err)
		return err
	}
	return nil
}

func GetBySourceId(sourceId string) LocalPackageItem {
	localPackageRoot := GetData(false)
	for _, item := range localPackageRoot.Packages {
		if item.SourceID == sourceId {
			return item
		}
	}
	return LocalPackageItem{}
}

func IsPackageInstalled(sourceId string) bool {
	localPackageRoot := GetData(false)
	for _, item := range localPackageRoot.Packages {
		if item.SourceID == sourceId {
			return true
		}
	}
	return false
}
