package zana

import (
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/spf13/cobra"
)

// displayPackageIDFromRegistryID converts an internal registry/source ID into
// the user-facing format used on the CLI.
//
// Internal IDs are currently of the form:
//
//	pkg:<provider>/<package-id>
//
// and are exposed to users as:
//
//	<provider>:<package-id>
func displayPackageIDFromRegistryID(sourceID string) string {
	if strings.HasPrefix(sourceID, "pkg:") {
		rest := strings.TrimPrefix(sourceID, "pkg:")
		parts := strings.SplitN(rest, "/", 2)
		if len(parts) == 2 {
			return parts[0] + ":" + parts[1]
		}
	}
	return sourceID
}

// newRegistryParserFn is an indirection for tests.
var newRegistryParserFn = registry_parser.NewDefaultRegistryParser

// packageIDCompletion provides shell completion for package IDs based on the
// locally available registry data.
func packageIDCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	parser := newRegistryParserFn()
	items := parser.GetData(false)

	completions := make([]string, 0, len(items))
	for _, item := range items {
		displayID := displayPackageIDFromRegistryID(strings.TrimSpace(item.Source.ID))
		if displayID == "" {
			continue
		}
		if toComplete == "" || strings.HasPrefix(displayID, toComplete) {
			completions = append(completions, displayID)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
