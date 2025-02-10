package updater

import (
	"strings"

	"github.com/charmbracelet/log"
)

type NPMProvider struct{}

// getPackageId returns the package ID from a given source
func (p *NPMProvider) getPackageId(sourceId string) string {
	sourceId = strings.TrimPrefix(sourceId, "pkg:")
	parts := strings.Split(sourceId, "/")
	return strings.Join(parts[1:], "/")
}

func (p *NPMProvider) Update(source string) {
	log.Info("Updating via NPM provider", "source", source)
}
