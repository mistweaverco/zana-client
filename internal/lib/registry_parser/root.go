package registry_parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/mistweaverco/zana-client/internal/lib/files"
)

type RegistryItemSource struct {
	ID string `json:"id"`
}

type RegistryItem struct {
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	Homepage    string             `json:"homepage"`
	Licenses    []string           `json:"licenses"`
	Languages   []string           `json:"languages"`
	Categories  []string           `json:"categories"`
	Source      RegistryItemSource `json:"source"`
	Bin         map[string]string  `json:"bin"`
}

type RegistryRoot []RegistryItem

var data RegistryRoot
var hasData bool

func GetData(force bool) RegistryRoot {
	if hasData && !force {
		return data
	}
	registryFile := files.GetAppRegistryFilePath()
	var registry RegistryRoot
	jsonFile, err := os.Open(registryFile)
	if err != nil {
		// Return empty registry instead of panicking when file doesn't exist
		data = RegistryRoot{}
		hasData = true
		return data
	}
	defer func() {
		if closeErr := jsonFile.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close registry file: %v\n", closeErr)
		}
	}()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		fmt.Printf("Warning: failed to read registry file: %v\n", err)
		return RegistryRoot{}
	}
	if err := json.Unmarshal(byteValue, &registry); err != nil {
		fmt.Printf("Warning: failed to parse registry file: %v\n", err)
		return RegistryRoot{}
	}
	// Sort the registry by name
	hasData = true
	data = registry
	sort.Slice(registry, func(i, j int) bool {
		return registry[i].Name < registry[j].Name
	})
	return registry
}

func GetBySourceId(sourceId string) RegistryItem {
	registryRoot := GetData(false)
	for _, item := range registryRoot {
		if item.Source.ID == sourceId {
			return item
		}
	}
	return RegistryItem{}
}

func GetLatestVersion(sourceId string) string {
	item := GetBySourceId(sourceId)
	return item.Version
}
