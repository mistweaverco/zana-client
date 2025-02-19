package updater

import "log"

type BitbucketProvider struct{}

// Install or update a package via the Bitbucket provider
func (g *BitbucketProvider) Install(source string, version string) {
	log.Println("Updating via Bitbucket provider", "source", source)
}
