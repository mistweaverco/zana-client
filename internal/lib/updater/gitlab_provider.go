package updater

import "github.com/charmbracelet/log"

type GitLabProvider struct{}

func (g *GitLabProvider) Update(source string) {
	log.Info("Updating via GitLab provider", "source", source)
}
