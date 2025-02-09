package updater

import "github.com/charmbracelet/log"

type GitHubProvider struct{}

func (g *GitHubProvider) Update(source string) {
	log.Info("Updating via GitHub provider", "source", source)
}
