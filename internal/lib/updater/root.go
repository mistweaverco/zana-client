package updater

import (
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/semver"
)

type Provider int

const (
	ProviderNPM Provider = iota
	ProviderPyPi
	ProviderUnsupported
)

var npmProvider NPMProvider = *NewProviderNPM()
var pypiProvider PyPiProvider = *NewProviderPyPi()

func detectProvider(sourceId string) Provider {
	var provider Provider
	switch {
	case strings.HasPrefix(sourceId, npmProvider.PREFIX):
		provider = ProviderNPM
	case strings.HasPrefix(sourceId, pypiProvider.PREFIX):
		provider = ProviderPyPi
	default:
		provider = ProviderUnsupported
	}
	return provider
}

// CheckIfUpdateIsAvailable checks if an update is available for a given package
// and returns a boolean indicating if an update is available and the latest version number
func CheckIfUpdateIsAvailable(localVersion string, remoteVersion string) (bool, string) {
	if semver.IsGreater(localVersion, remoteVersion) {
		return true, remoteVersion
	}
	return false, ""
}

func Install(sourceId string, version string) bool {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderNPM:
		return npmProvider.Install(sourceId, version)
	case ProviderPyPi:
		return pypiProvider.Install(sourceId, version)
	case ProviderUnsupported:
		// Unsupported provider
	}
	return false
}

func Remove(sourceId string) bool {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderNPM:
		return npmProvider.Remove(sourceId)
	case ProviderPyPi:
		return pypiProvider.Remove(sourceId)
	case ProviderUnsupported:
		// Unsupported provider
	}
	return false
}
