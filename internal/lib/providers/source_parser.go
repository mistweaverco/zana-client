package providers

import "strings"

type SourceParser struct{}

func (sp *SourceParser) ParseSource(source string) (string, string) {
	source = strings.TrimPrefix(source, "pkg:")
	parts := strings.Split(source, "/")
	provider := parts[0]
	packageId := parts[1]
	return provider, packageId
}