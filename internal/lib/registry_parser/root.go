package registry_parser

import (
	"encoding/json"
	"io"
	"os"

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
	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &registry)
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
