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

// RegistryItemSourceAssetFile can be a string or an array of strings
type RegistryItemSourceAssetFile struct {
	value interface{}
}

func (f *RegistryItemSourceAssetFile) UnmarshalJSON(data []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		f.value = str
		return nil
	}
	// Try array
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		f.value = arr
		return nil
	}
	return fmt.Errorf("cannot unmarshal file: expected string or array")
}

func (f *RegistryItemSourceAssetFile) String() string {
	if str, ok := f.value.(string); ok {
		return str
	}
	if arr, ok := f.value.([]string); ok && len(arr) > 0 {
		return arr[0] // Return first file if it's an array
	}
	return ""
}

func (f *RegistryItemSourceAssetFile) IsArray() bool {
	_, ok := f.value.([]string)
	return ok
}

func (f *RegistryItemSourceAssetFile) GetArray() []string {
	if arr, ok := f.value.([]string); ok {
		return arr
	}
	if str, ok := f.value.(string); ok {
		return []string{str}
	}
	return nil
}

type RegistryItemSourceAsset struct {
	Target interface{}                 `json:"target"` // Can be string or []string
	File   RegistryItemSourceAssetFile `json:"file"`
	Bin    interface{}                 `json:"bin,omitempty"` // Can be string or map[string]string
}

// RegistryItemSourceAssetList is a custom type that can unmarshal both a single object and an array
type RegistryItemSourceAssetList []RegistryItemSourceAsset

// UnmarshalJSON implements custom JSON unmarshaling to handle both single object and array formats
func (a *RegistryItemSourceAssetList) UnmarshalJSON(data []byte) error {
	// Handle null or empty values
	if len(data) == 0 || string(data) == "null" {
		*a = nil
		return nil
	}

	// Try to unmarshal as array first
	var arr []RegistryItemSourceAsset
	if err := json.Unmarshal(data, &arr); err == nil {
		*a = arr
		return nil
	}

	// If that fails, try as a single object
	var obj RegistryItemSourceAsset
	if err := json.Unmarshal(data, &obj); err == nil {
		*a = []RegistryItemSourceAsset{obj}
		return nil
	}

	return fmt.Errorf("cannot unmarshal asset: expected array or object, got: %s", string(data))
}

type RegistryItemSourceDownloadFile struct {
	Target interface{}       `json:"target"` // Can be string or []string
	Files  map[string]string `json:"files"`  // Map of filename -> URL
	Bin    string            `json:"bin,omitempty"`
}

type RegistryItemSourceDownloadList []RegistryItemSourceDownloadFile

// UnmarshalJSON implements custom JSON unmarshaling to handle both single object and array formats
func (d *RegistryItemSourceDownloadList) UnmarshalJSON(data []byte) error {
	// Handle null or empty values
	if len(data) == 0 || string(data) == "null" {
		*d = nil
		return nil
	}

	// Try to unmarshal as array first
	var arr []RegistryItemSourceDownloadFile
	if err := json.Unmarshal(data, &arr); err == nil {
		*d = arr
		return nil
	}

	// If that fails, try as a single object
	var obj RegistryItemSourceDownloadFile
	if err := json.Unmarshal(data, &obj); err == nil {
		*d = []RegistryItemSourceDownloadFile{obj}
		return nil
	}

	return fmt.Errorf("cannot unmarshal download: expected array or object, got: %s", string(data))
}

type RegistryItemSource struct {
	ID       string                         `json:"id"`
	Asset    RegistryItemSourceAssetList    `json:"asset,omitempty"`
	Download RegistryItemSourceDownloadList `json:"download,omitempty"`
}

type RegistryItem struct {
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Description string             `json:"description"`
	Homepage    string             `json:"homepage"`
	Licenses    []string           `json:"licenses"`
	Languages   []string           `json:"languages"`
	Categories  []string           `json:"categories"`
	Aliases     []string           `json:"aliases,omitempty"`
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

// GetByNameOrAlias finds a registry item by its name or any of its aliases.
// It prioritizes exact name matches over alias matches.
func (rp *RegistryParser) GetByNameOrAlias(name string) RegistryItem {
	registryRoot := rp.GetData(false)

	// First pass: check for exact name matches (prioritize these)
	for _, item := range registryRoot {
		if item.Name == name {
			return item
		}
	}

	// Second pass: check for alias matches only if no name match was found
	for _, item := range registryRoot {
		for _, alias := range item.Aliases {
			if alias == name {
				return item
			}
		}
	}

	return RegistryItem{}
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
