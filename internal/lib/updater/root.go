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

func GetLatestReleaseVersionNumber(sourceId string) string {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderGitHub:
		return gitHubProvider.GetLatestReleaseVersionNumber(sourceId)
	case ProviderUnsupported:
		// Unsupported provider
		return ""
	default:
		return ""
	}
}

// CheckIfUpdateIsAvailable checks if an update is available for a given package
// and returns a boolean indicating if an update is available and the latest version number
func CheckIfUpdateIsAvailable(version string, sourceId string) (bool, string) {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderGitHub:
		latestVersion := gitHubProvider.GetLatestReleaseVersionNumber(sourceId)
		if semver.IsGreater(version, latestVersion) {
			return true, latestVersion
		}
		return false, latestVersion
	case ProviderUnsupported:
		// Unsupported provider
		return false, ""
	default:
		return false, ""
	}
}

func Update(sourceId string) {
	provider := detectProvider(sourceId)
	switch provider {
	case ProviderGitHub:
		gitHubProvider.Update(sourceId)
	case ProviderGitLab:
		gitLabProvider.Update(sourceId)
	case ProviderBitbucket:
		bitbucketProvider.Update(sourceId)
	case ProviderNPM:
		npmProvider.Update(sourceId)
	case ProviderUnsupported:
		// Unsupported provider
	}
}
