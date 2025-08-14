package registry_parser

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/mistweaverco/zana-client/internal/lib/files"
)

// FileReader interface for dependency injection in tests
type FileReader interface {
	ReadFile(filename string) ([]byte, error)
}

// defaultFileReader implements FileReader using the standard library
type defaultFileReader struct{}

func (d *defaultFileReader) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
}

// RegistryParser handles parsing of registry data
type RegistryParser struct {
	fileReader FileReader
	data       RegistryRoot
	hasData    bool
}

// NewRegistryParser creates a new RegistryParser instance
func NewRegistryParser(fileReader FileReader) *RegistryParser {
	return &RegistryParser{
		fileReader: fileReader,
		data:       RegistryRoot{},
		hasData:    false,
	}
}

// NewDefaultRegistryParser creates a RegistryParser with default dependencies
func NewDefaultRegistryParser() *RegistryParser {
	return NewRegistryParser(&defaultFileReader{})
}

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

// GetData retrieves registry data, optionally forcing a refresh
func (rp *RegistryParser) GetData(force bool) RegistryRoot {
	if rp.hasData && !force {
		return rp.data
	}

	// Try to load from the default registry file path
	// This maintains backward compatibility with the old implementation
	registryFile := files.GetAppRegistryFilePath()
	if err := rp.LoadFromFile(registryFile); err != nil {
		// If file loading fails, return empty data
		rp.data = RegistryRoot{}
		rp.hasData = true
	}

	return rp.data
}

// GetBySourceId finds a registry item by its source ID
func (rp *RegistryParser) GetBySourceId(sourceId string) RegistryItem {
	registryRoot := rp.GetData(false)
	for _, item := range registryRoot {
		if item.Source.ID == sourceId {
			return item
		}
	}
	return RegistryItem{}
}

// GetLatestVersion gets the latest version for a given source ID
func (rp *RegistryParser) GetLatestVersion(sourceId string) string {
	item := rp.GetBySourceId(sourceId)
	return item.Version
}

// LoadFromBytes loads registry data from JSON bytes
func (rp *RegistryParser) LoadFromBytes(data []byte) error {
	var registry RegistryRoot
	if err := json.Unmarshal(data, &registry); err != nil {
		return fmt.Errorf("failed to parse registry data: %w", err)
	}

	// Sort the registry by name
	sort.Slice(registry, func(i, j int) bool {
		return registry[i].Name < registry[j].Name
	})

	rp.data = registry
	rp.hasData = true
	return nil
}

// LoadFromFile loads registry data from a file
func (rp *RegistryParser) LoadFromFile(filename string) error {
	data, err := rp.fileReader.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read registry file: %w", err)
	}

	return rp.LoadFromBytes(data)
}

// GetDataForTesting returns the current data (useful for testing)
func (rp *RegistryParser) GetDataForTesting() RegistryRoot {
	return rp.data
}

// HasDataForTesting returns whether data has been loaded (useful for testing)
func (rp *RegistryParser) HasDataForTesting() bool {
	return rp.hasData
}
