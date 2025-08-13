package zana

import (
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
)

// LocalPackagesProvider defines the interface for getting local packages data
type LocalPackagesProvider interface {
	GetData(force bool) local_packages_parser.LocalPackageRoot
}

// RegistryProvider defines the interface for getting registry data
type RegistryProvider interface {
	GetData(force bool) []registry_parser.RegistryItem
	GetLatestVersion(sourceID string) string
}

// UpdateChecker defines the interface for checking if updates are available
type UpdateChecker interface {
	CheckIfUpdateIsAvailable(currentVersion, latestVersion string) (bool, string)
}

// FileDownloader defines the interface for downloading files
type FileDownloader interface {
	DownloadAndUnzipRegistry() error
}
