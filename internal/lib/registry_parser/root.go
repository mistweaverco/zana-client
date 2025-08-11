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
	ID    string `json:"id"`
	Asset struct {
		Target string `json:"target"`
		File   string `json:"file"`
	} `json:"asset,omitempty"`
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
		panic(err)
	}
	defer func() {
		if closeErr := jsonFile.Close(); closeErr != nil {
			panic(fmt.Sprintf("failed to close registry file: %v", closeErr))
		}
	}()
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read registry file: %v", err))
	}
	if err := json.Unmarshal(byteValue, &registry); err != nil {
		panic(fmt.Sprintf("failed to parse registry file: %v", err))
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
