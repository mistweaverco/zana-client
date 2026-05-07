package providers

import "strings"

// integrations requested by the CLI layer (e.g. ["neovim"]).
var requestedIntegrations []string

func SetRequestedIntegrations(integrations []string) {
	requestedIntegrations = nil
	for _, i := range integrations {
		i = strings.ToLower(strings.TrimSpace(i))
		if i == "" {
			continue
		}
		requestedIntegrations = append(requestedIntegrations, i)
	}
}

func integrationEnabled(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	for _, i := range requestedIntegrations {
		if i == name {
			return true
		}
	}
	return false
}

