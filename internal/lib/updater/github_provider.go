package updater

import (
	"strings"

	"github.com/charmbracelet/log"
)

type GitHubProvider struct{}

// getPackageId returns the package ID from a given source
func (g *GitHubProvider) getPackageId(sourceId string) string {
	sourceId = strings.TrimPrefix(sourceId, "pkg:")
	parts := strings.Split(sourceId, "/")
	return strings.Join(parts[1:], "/")
}

// Install or update a package via the GitHub provider
func (g *GitHubProvider) Install(sourceId string, version string) {
	log.Info("Updating via GitHub provider", "source", sourceId)
}
