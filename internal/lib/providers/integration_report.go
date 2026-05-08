package providers

import "strings"

// integrationReports is a best-effort side-channel for the CLI to display
// where integrations installed things.
// Key format: "<sourceID>@<version>"
var integrationReports = map[string][]string{}

func integrationReportKey(sourceID, version string) string {
	return strings.TrimSpace(sourceID) + "@" + strings.TrimSpace(version)
}

func AddIntegrationReportLine(sourceID, version, line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	k := integrationReportKey(sourceID, version)
	integrationReports[k] = append(integrationReports[k], line)
}

// ConsumeIntegrationReport exposes per-install integration messages for the CLI.
// It is best-effort; when no integrations ran, it returns nil/empty.
func ConsumeIntegrationReport(sourceID, version string) []string {
	k := integrationReportKey(sourceID, version)
	lines := integrationReports[k]
	delete(integrationReports, k)
	return lines
}
