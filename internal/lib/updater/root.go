package updater

import (
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/semver"
)

type Provider int

const (
	ProviderGitHub Provider = iota
	ProviderGitLab
	ProviderBitbucket
	ProviderNPM
	ProviderUnsupported
)

var gitHubProvider GitHubProvider = GitHubProvider{}
var gitLabProvider GitLabProvider = GitLabProvider{}
var bitbucketProvider BitbucketProvider = BitbucketProvider{}
var npmProvider NPMProvider = NPMProvider{}

func detectProvider(sourceId string) Provider {
	var provider Provider
	switch {
	case strings.HasPrefix(sourceId, "pkg:github"):
		provider = ProviderGitHub
	case strings.HasPrefix(sourceId, "pkg:gitlab"):
		provider = ProviderGitLab
	case strings.HasPrefix(sourceId, "pkg:bitbucket"):
		provider = ProviderBitbucket
	case strings.HasPrefix(sourceId, "pkg:npm"):
		provider = ProviderNPM
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
	case ProviderGitHub:
		gitHubProvider.Install(sourceId, version)
	case ProviderGitLab:
		gitLabProvider.Install(sourceId, version)
	case ProviderBitbucket:
		bitbucketProvider.Install(sourceId, version)
	case ProviderNPM:
		return npmProvider.Install(sourceId, version)
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
	case ProviderUnsupported:
		// Unsupported provider
	}
	return false
}
