package updater

import "strings"

type SourceParser struct{}

func (sp *SourceParser) getLatestReleaseVersionNumber(source string) string {
	provider, packageId := sp.ParseSource(source)
	switch provider {
	case "github":
		return gitHubProvider.GetLatestReleaseVersionNumber(packageId)
	default:
		return ""
	}
}

func (sp *SourceParser) ParseSource(source string) (string, string) {
	source = strings.TrimPrefix(source, "pkg:")
	parts := strings.Split(source, "/")
	provider := parts[0]
	packageId := parts[1]
	return provider, packageId
}
