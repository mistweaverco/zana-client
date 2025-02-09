package updater

import "github.com/charmbracelet/log"

type BitbucketProvider struct{}

func (g *BitbucketProvider) Update(source string) {
	log.Info("Updating via Bitbucket provider", "source", source)
}
