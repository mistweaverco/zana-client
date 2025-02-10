package updater

import "github.com/charmbracelet/log"

type BitbucketProvider struct{}

// Install or update a package via the Bitbucket provider
func (g *BitbucketProvider) Install(source string, version string) {
	log.Info("Updating via Bitbucket provider", "source", source)
}
