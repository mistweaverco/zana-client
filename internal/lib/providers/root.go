package providers

import (
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/log"
	"github.com/mistweaverco/zana-client/internal/lib/semver"
)

type Provider int

const (
	ProviderNPM Provider = iota
	ProviderPyPi
	ProviderGolang
	ProviderCargo
	ProviderUnsupported
)

var Logger = log.NewLogger()

// Global factory instance - can be replaced for testing
var globalFactory ProviderFactory = &DefaultProviderFactory{}

// SetProviderFactory allows setting a custom factory for testing
func SetProviderFactory(factory ProviderFactory) {
	globalFactory = factory
}

// ResetProviderFactory resets to the default factory
func ResetProviderFactory() {
	globalFactory = &DefaultProviderFactory{}
}

// Get providers from factory
func getNPMProvider() PackageManager {
	return globalFactory.CreateNPMProvider()
}

func getPyPIProvider() PackageManager {
	return globalFactory.CreatePyPIProvider()
}

func getGolangProvider() PackageManager {
	return globalFactory.CreateGolangProvider()
}

func getCargoProvider() PackageManager {
	return globalFactory.CreateCargoProvider()
}

// AvailableProviders lists all provider names supported by Zana
var AvailableProviders = []string{
	"npm",
	"pypi",
	"golang",
	"cargo",
}

// IsSupportedProvider returns true if the given provider name is supported
func IsSupportedProvider(name string) bool {
	for _, p := range AvailableProviders {
		if p == name {
			return true
		}
	}
	return false
}

// normalizePackageID converts a package ID from legacy format (pkg:provider/pkg)
// to the new format (provider:pkg), or returns it unchanged if already in new format.
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

// extractProviderAndPackage extracts provider and package name from a source ID.
// Supports both legacy (pkg:provider/pkg) and new (provider:pkg) formats.
func extractProviderAndPackage(sourceID string) (string, string) {
	normalized := normalizePackageID(sourceID)
	parts := strings.SplitN(normalized, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func detectProvider(sourceId string) Provider {
	normalized := normalizePackageID(sourceId)
	providerName, _ := extractProviderAndPackage(normalized)
	if providerName == "" {
		return ProviderUnsupported
	}

	switch strings.ToLower(providerName) {
	case "npm":
		return ProviderNPM
	case "pypi":
		return ProviderPyPi
	case "golang":
		return ProviderGolang
	case "cargo":
		return ProviderCargo
	default:
		return ProviderUnsupported
	}
}

// CheckIfUpdateIsAvailable checks if an update is available for a given package
// and returns a boolean indicating if an update is available and the latest version number
func CheckIfUpdateIsAvailable(localVersion string, remoteVersion string) (bool, string) {
	if semver.IsGreater(localVersion, remoteVersion) {
		return true, remoteVersion
	}
	return false, ""
}

func SyncAll() {
	npmProvider := getNPMProvider()
	if npm, ok := npmProvider.(*NPMProvider); ok {
		npm.Sync()
	}

	pypiProvider := getPyPIProvider()
	if pypi, ok := pypiProvider.(*PyPiProvider); ok {
		pypi.Sync()
	}

	golangProvider := getGolangProvider()
	if golang, ok := golangProvider.(*GolangProvider); ok {
		golang.Sync()
	}

	cargoProvider := getCargoProvider()
	if cargo, ok := cargoProvider.(*CargoProvider); ok {
		cargo.Sync()
	}
}

func Install(sourceId string, version string) bool {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderNPM:
		return getNPMProvider().Install(sourceId, version)
	case ProviderPyPi:
		return getPyPIProvider().Install(sourceId, version)
	case ProviderGolang:
		return getGolangProvider().Install(sourceId, version)
	case ProviderCargo:
		return getCargoProvider().Install(sourceId, version)
	case ProviderUnsupported:
		// Unsupported provider
	}
	return false
}

func Remove(sourceId string) bool {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderNPM:
		return getNPMProvider().Remove(sourceId)
	case ProviderPyPi:
		return getPyPIProvider().Remove(sourceId)
	case ProviderGolang:
		return getGolangProvider().Remove(sourceId)
	case ProviderCargo:
		return getCargoProvider().Remove(sourceId)
	case ProviderUnsupported:
		// Unsupported provider
	}
	return false
}

func Update(sourceId string) bool {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderNPM:
		return getNPMProvider().Update(sourceId)
	case ProviderPyPi:
		return getPyPIProvider().Update(sourceId)
	case ProviderGolang:
		return getGolangProvider().Update(sourceId)
	case ProviderCargo:
		return getCargoProvider().Update(sourceId)
	case ProviderUnsupported:
		// Unsupported provider
	}
	return false
}
