package updater

import (
	"strings"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/shell_out"
)

type NPMProvider struct{}

// getPackageId returns the package ID from a given source
func (p *NPMProvider) getPackageId(sourceId string) string {
	sourceId = strings.TrimPrefix(sourceId, "pkg:")
	parts := strings.Split(sourceId, "/")
	return strings.Join(parts[1:], "/")
}

var packagesPath = files.EnsureDirExists(files.GetAppPackagesPath() + files.PS + "npm")

// Install or update a package via the NPM provider
func (p *NPMProvider) Install(source string, version string) {
	shell_out.ShellOut("npm", "install", "--prefix", packagesPath, p.getPackageId(source))
	local_packages_parser.AddLocalPackage(source, version)
}
