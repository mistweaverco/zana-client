package providers

import (
	"log"
)

type GitHubProvider struct{}

// Install or update a package via the GitHub provider
func (g *GitHubProvider) Install(sourceId string, version string) {
	log.Println("Updating via GitHub provider", "source", sourceId)
}
