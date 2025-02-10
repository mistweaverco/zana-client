package local_packages_parser

import (
	"encoding/json"
	"io"
	"os"

	"github.com/mistweaverco/zana-client/internal/lib/files"
)

type LocalPackageItem struct {
	SourceID string `json:"source_id"`
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
	defer jsonFile.Close()
	byteValue, _ := io.ReadAll(jsonFile)
	json.Unmarshal(byteValue, &localPackageRoot)
	hasData = true
	data = localPackageRoot
	return data
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
