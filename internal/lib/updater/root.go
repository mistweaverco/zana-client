package updater

import "strings"

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

func detectProvider(source string) Provider {
	var provider Provider
	switch {
	case strings.HasPrefix(source, "pkg:github"):
		provider = ProviderGitHub
	case strings.HasPrefix(source, "pkg:gitlab"):
		provider = ProviderGitLab
	case strings.HasPrefix(source, "pkg:bitbucket"):
		provider = ProviderBitbucket
	case strings.HasPrefix(source, "pkg:npm"):
		provider = ProviderNPM
	default:
		provider = ProviderUnsupported
	}
	return provider
}

func Update(source string) {
	provider := detectProvider(source)
	switch provider {
	case ProviderGitHub:
		gitHubProvider.Update(source)
	case ProviderGitLab:
		gitLabProvider.Update(source)
	case ProviderBitbucket:
		bitbucketProvider.Update(source)
	case ProviderNPM:
		npmProvider.Update(source)
	case ProviderUnsupported:
		// Unsupported provider
	}
}
