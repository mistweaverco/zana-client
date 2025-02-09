package updater

import "github.com/charmbracelet/log"

type NPMProvider struct{}

func (g *NPMProvider) Update(source string) {
	log.Info("Updating via NPM provider", "source", source)
}
