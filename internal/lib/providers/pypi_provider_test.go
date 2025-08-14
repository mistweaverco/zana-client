package providers

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/stretchr/testify/assert"
)

// helper to set ZANA_HOME to a temp dir
func withTempZanaHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ZANA_HOME", dir)
	// Ensure dirs initialized
	_ = files.GetAppDataPath()
	_ = files.GetAppBinPath()
	_ = files.GetAppPackagesPath()
	return dir
}

func writeRegistry(t *testing.T, items []registry_parser.RegistryItem) {
	t.Helper()
	path := files.GetAppRegistryFilePath()
	data, err := json.Marshal(items)
	assert.NoError(t, err)
	err = os.WriteFile(path, data, 0644)
	assert.NoError(t, err)
}

func TestNPMProviderBasicFlows(t *testing.T) {
	_ = withTempZanaHome(t)

	// stub shell helpers
	oldOut := npmShellOut
	oldCap := npmShellOutCapture
	oldCreate := npmCreate
	npmShellOut = func(cmd string, args []string, dir string, env []string) (int, error) { return 0, nil }
	npmShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		return 0, "1.2.3\n", nil
	}
	npmCreate = func(name string) (*os.File, error) {
		_ = os.MkdirAll(filepath.Dir(name), 0755)
		return os.Create(name)
	}
	t.Cleanup(func() { npmShellOut = oldOut; npmShellOutCapture = oldCap; npmCreate = oldCreate })

	p := NewProviderNPM()
	assert.Equal(t, "npm", p.PROVIDER_NAME)
	assert.Equal(t, "pkg:npm/", p.PREFIX)

	// ensure provider packages dir exists
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// getRepo
	assert.Equal(t, "eslint", p.getRepo("pkg:npm/eslint"))

	// generatePackageJSON with no packages
	ok := p.generatePackageJSON()
	assert.False(t, ok)

	// add a local npm package and generate again
	_ = local_packages_parser.AddLocalPackage("pkg:npm/eslint", "1.0.0")
	ok = p.generatePackageJSON()
	assert.True(t, ok)

	// create node_modules package.json for bin
	nm := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "eslint")
	_ = os.MkdirAll(nm, 0755)
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	// create actual bin target to avoid chmod errors
	assert.NoError(t, os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin", "eslint"), []byte(""), 0755))
	pkgJSON := `{"name":"eslint","version":"1.0.0","bin":{"eslint":"./bin/eslint.js"}}`
	assert.NoError(t, os.WriteFile(filepath.Join(nm, "package.json"), []byte(pkgJSON), 0644))

	// isPackageInstalled true/false
	assert.True(t, p.isPackageInstalled("eslint", "1.0.0"))
	assert.False(t, p.isPackageInstalled("eslint", "2.0.0"))

	// createPackageSymlinks
	assert.NoError(t, p.createPackageSymlinks("eslint"))

	// removePackageSymlinks
	assert.NoError(t, p.removePackageSymlinks("eslint"))

	// getInstalledPackagesFromLock
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	lockData := `{"dependencies":{"eslint":{"version":"1.0.0"}}}`
	assert.NoError(t, os.WriteFile(lock, []byte(lockData), 0644))
	inst := p.getInstalledPackagesFromLock(lock)
	assert.Equal(t, "1.0.0", inst["eslint"])
	// ensure lock is newer than package.json so fast-path is taken
	pkgPath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	assert.NoError(t, os.WriteFile(pkgPath, []byte("{}"), 0644))
	now := time.Now()
	_ = os.Chtimes(lock, now.Add(1*time.Hour), now.Add(1*time.Hour))

	// tryNpmCi: lock exists and stub returns success
	assert.True(t, p.tryNpmCi())
	// failure path
	npmShellOut = func(cmd string, args []string, dir string, env []string) (int, error) { return 1, nil }
	assert.False(t, p.tryNpmCi())
	// reset to success before Sync and later flows
	npmShellOut = func(cmd string, args []string, dir string, env []string) (int, error) { return 0, nil }

	// Sync reaches install individually path (node_modules has correct content, installs ok)
	ok = p.Sync()
	assert.True(t, ok)

	// Install with latest
	ok = p.Install("pkg:npm/eslint", "latest")
	assert.True(t, ok)

	// Update
	ok = p.Update("pkg:npm/eslint")
	assert.True(t, ok)

	// Clean
	ok = p.Clean()
	assert.True(t, ok)

	// Remove
	ok = p.Remove("pkg:npm/eslint")
	assert.True(t, ok)

	// hasPackageJSONChanged scenarios after flows
	// now make package newer to toggle path
	_ = os.Chtimes(pkgPath, now.Add(2*time.Hour), now.Add(2*time.Hour))
	assert.True(t, p.hasPackageJSONChanged())
}

func TestNPMCustomBinFieldUnmarshal(t *testing.T) {
	var cbf CustomBinField
	// string case
	err := cbf.UnmarshalJSON([]byte(`"./bin/cli.js"`))
	assert.NoError(t, err)
	// map case
	err = cbf.UnmarshalJSON([]byte(`{"foo":"./bin/foo.js"}`))
	assert.NoError(t, err)
	// invalid type
	err = cbf.UnmarshalJSON([]byte(`123`))
	assert.Error(t, err)
}

func TestPyPiProviderBasicFlows(t *testing.T) {
	_ = withTempZanaHome(t)

	// stub shell
	oldOut := pipShellOut
	oldCap := pipShellOutCapture
	oldCreate := pipCreate
	pipShellOut = func(cmd string, args []string, dir string, env []string) (int, error) { return 0, nil }
	pipShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		return 0, "package==1.0.0\n", nil
	}
	pipCreate = func(name string) (*os.File, error) {
		_ = os.MkdirAll(filepath.Dir(name), 0755)
		return os.Create(name)
	}
	t.Cleanup(func() { pipShellOut = oldOut; pipShellOutCapture = oldCap; pipCreate = oldCreate })

	p := NewProviderPyPi()
	assert.Equal(t, "pypi", p.PROVIDER_NAME)
	assert.Equal(t, "pkg:pypi/", p.PREFIX)

	// ensure provider packages dir exists
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// add local package and registry mapping
	_ = local_packages_parser.AddLocalPackage("pkg:pypi/black", "1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "black", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/black"},
		Bin: map[string]string{"black": "black"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)

	// generate requirements
	ok := p.generateRequirementsTxt()
	assert.True(t, ok)

	// getInstalledPackages parses freeze output
	installed := p.getInstalledPackages()
	assert.Equal(t, "1.0.0", installed["package"])

	// create wrappers
	assert.NoError(t, p.createWrappers())

	// createPythonWrapperForCommand with empty command
	err := p.createPythonWrapperForCommand("", filepath.Join(files.GetAppBinPath(), "noop"))
	assert.Error(t, err)

	// Simulate site-packages and entry_points for removeBin
	site := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages", "black-1.0.0.dist-info")
	_ = os.MkdirAll(site, 0755)
	ep := "[console_scripts]\nblack=black:main\n"
	assert.NoError(t, os.WriteFile(filepath.Join(site, "entry_points.txt"), []byte(ep), 0644))
	// create a bin file to remove
	binFile := filepath.Join(p.APP_PACKAGES_DIR, "bin", "black")
	_ = os.MkdirAll(filepath.Dir(binFile), 0755)
	assert.NoError(t, os.WriteFile(binFile, []byte("#!/bin/sh\n"), 0755))
	assert.NoError(t, p.removeBin("pkg:pypi/black"))

	// getLatestVersion parses pip index output
	pipShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		return 0, "Available versions: 2.0.0, 1.0.0, 0.1", nil
	}
	v, err := p.getLatestVersion("black")
	assert.NoError(t, err)
	assert.Equal(t, "2.0.0", v)

	// Install latest
	ok = p.Install("pkg:pypi/black", "latest")
	assert.True(t, ok)

	// Update
	ok = p.Update("pkg:pypi/black")
	assert.True(t, ok)

	// Remove (no wrappers existing should still succeed)
	ok = p.Remove("pkg:pypi/black")
	assert.True(t, ok)

	// removeAllWrappers when wrappers exist
	_ = os.WriteFile(filepath.Join(files.GetAppBinPath(), "black"), []byte(""), 0755)
	assert.NoError(t, p.removeAllWrappers())

	// Clean
	ok = p.Clean()
	assert.True(t, ok)
}

func TestPyPiReadPackageInfo(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	// create a fake package dir with .dist-info and METADATA
	base := filepath.Join(p.APP_PACKAGES_DIR, "site")
	infoDir := filepath.Join(base, "foo-1.2.3.dist-info")
	_ = os.MkdirAll(infoDir, 0755)
	metadata := "Name: foo\nVersion: 1.2.3\n"
	assert.NoError(t, os.WriteFile(filepath.Join(infoDir, "METADATA"), []byte(metadata), 0644))
	info, err := p.readPackageInfo(base)
	assert.NoError(t, err)
	assert.Equal(t, "foo", info.Name)
	assert.Equal(t, "1.2.3", info.Version)
}

func TestGolangProviderBasicFlows(t *testing.T) {
	_ = withTempZanaHome(t)

	oldOut := goShellOut
	oldCap := goShellOutCapture
	oldCreate := goCreate
	goShellOut = func(cmd string, args []string, dir string, env []string) (int, error) { return 0, nil }
	goShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		return 0, "module x v1.0.0 v2.0.0", nil
	}
	goCreate = func(name string) (*os.File, error) {
		_ = os.MkdirAll(filepath.Dir(name), 0755)
		return os.Create(name)
	}
	t.Cleanup(func() { goShellOut = oldOut; goShellOutCapture = oldCap; goCreate = oldCreate })

	p := NewProviderGolang()
	assert.Equal(t, "golang", p.PROVIDER_NAME)
	assert.Equal(t, "pkg:golang/", p.PREFIX)

	// add local package and registry mapping
	_ = local_packages_parser.AddLocalPackage("pkg:golang/github.com/acme/tool", "1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/acme/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)

	// precreate binary to allow createSymlink to succeed
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(gobin, "tool"), []byte("#!/bin/sh\n"), 0755))

	// Sync should run and create symlink
	ok := p.Sync()
	assert.True(t, ok)

	// Update path
	ok = p.Update("pkg:golang/github.com/acme/tool")
	assert.True(t, ok)

	// Install latest
	ok = p.Install("pkg:golang/github.com/acme/tool", "latest")
	assert.True(t, ok)

	// Remove
	ok = p.Remove("pkg:golang/github.com/acme/tool")
	assert.True(t, ok)

	// Clean
	ok = p.Clean()
	assert.True(t, ok)

	// create and remove symlink explicitly: ensure binary exists first
	gobin2 := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin2, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(gobin2, "tool"), []byte(""), 0755))
	assert.NoError(t, p.createSymlink("pkg:golang/github.com/acme/tool"))
	// verify symlink exists
	if _, e := os.Lstat(filepath.Join(files.GetAppBinPath(), "tool")); e != nil {
		t.Fatalf("expected symlink to exist: %v", e)
	}
	assert.NoError(t, p.removeSymlink("pkg:golang/github.com/acme/tool"))
}

func TestCargoProviderBasicFlows(t *testing.T) {
	_ = withTempZanaHome(t)

	oldOut := cargoShellOut
	oldCap := cargoShellOutCapture
	oldHas := cargoHasCommand
	cargoShellOut = func(cmd string, args []string, dir string, env []string) (int, error) { return 0, nil }
	cargoShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		if len(args) > 0 && args[0] == "search" {
			return 0, "mycrate = \"1.2.3\"", nil
		}
		return 0, "mycrate v1.0.0: installed binary\n", nil
	}
	cargoHasCommand = func(command string, args []string, env []string) bool { return true }
	t.Cleanup(func() { cargoShellOut = oldOut; cargoShellOutCapture = oldCap; cargoHasCommand = oldHas })

	p := NewProviderCargo()
	assert.Equal(t, "cargo", p.PROVIDER_NAME)
	assert.Equal(t, "pkg:cargo/", p.PREFIX)

	// ensure provider packages dir exists
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// add local package
	_ = local_packages_parser.AddLocalPackage("pkg:cargo/mycrate", "latest")

	// create cargo bin dir with a binary for symlink creation
	cargoBin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cargoBin, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(cargoBin, "mycrate"), []byte(""), 0755))

	// create a symlink in zana bin that points to cargo bin for removeAllSymlinks
	zbin := files.GetAppBinPath()
	sl := filepath.Join(zbin, "mycrate")
	_ = os.Symlink(filepath.Join(cargoBin, "mycrate"), sl)

	// getInstalledCrates
	crates := p.getInstalledCrates()
	assert.Equal(t, "1.0.0", crates["mycrate"]) // from stubbed capture

	// Sync
	ok := p.Sync()
	assert.True(t, ok)

	// Install latest
	ok = p.Install("pkg:cargo/mycrate", "latest")
	assert.True(t, ok)

	// Update
	ok = p.Update("pkg:cargo/mycrate")
	assert.True(t, ok)

	// Remove
	ok = p.Remove("pkg:cargo/mycrate")
	assert.True(t, ok)

	// Clean
	ok = p.Clean()
	assert.True(t, ok)

	// Remove again on missing to hit uninstall non-critical path
	ok = p.Remove("pkg:cargo/mycrate")
	assert.True(t, ok)
}

func TestFactoriesAndGithubProvider(t *testing.T) {
	// Default factory creates providers
	f := &DefaultProviderFactory{}
	assert.NotNil(t, f.CreateNPMProvider())
	assert.NotNil(t, f.CreatePyPIProvider())
	assert.NotNil(t, f.CreateGolangProvider())
	assert.NotNil(t, f.CreateCargoProvider())

	// MockPackageManager getLatestVersion default path
	m := &MockPackageManager{}
	ver, err := m.getLatestVersion("anything")
	assert.NoError(t, err)
	assert.Equal(t, "", ver)

	// GitHubProvider Install
	g := &GitHubProvider{}
	g.Install("pkg:github/owner/repo", "latest")
}
