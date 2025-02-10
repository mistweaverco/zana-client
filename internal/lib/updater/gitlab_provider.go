package updater

import "github.com/charmbracelet/log"

type GitLabProvider struct{}

// Install or update a package via the GitLab provider
func (g *GitLabProvider) Install(source string, version string) {
	log.Info("Updating via GitLab provider", "source", source)
}
