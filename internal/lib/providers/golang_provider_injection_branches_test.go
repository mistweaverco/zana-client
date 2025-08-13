package providers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/stretchr/testify/assert"
)

func TestGolangGeneratePackageJSON_CloseWarningAndEncodeError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Ensure there is at least one package so found == true
	_ = lppGoAdd("pkg:golang/github.com/acme/tool", "v1.0.0")

	// 1) Trigger close warning path
	oldCreate := goCreate
	oldClose := goClose
	goCreate = os.Create
	goClose = func(f *os.File) error { return errors.New("close") }
	assert.True(t, p.generatePackageJSON())
	goClose = oldClose

	// 2) Trigger encode error path by returning a closed file handle
	goCreate = func(path string) (*os.File, error) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
		return f, nil
	}
	assert.False(t, p.generatePackageJSON())
	goCreate = oldCreate
}

func TestGolangCreateSymlink_RemovesExistingAndSymlinkError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// registry with bin
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "v1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/acme/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.GetData(true)

	// ensure golang bin contains the binary
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(gobin, "tool"), []byte(""), 0755))

	// Prepare an existing symlink in zana bin to exercise removal path
	zbin := files.GetAppBinPath()
	_ = os.MkdirAll(zbin, 0755)
	pre := filepath.Join(zbin, "tool")
	_ = os.Symlink(filepath.Join(gobin, "tool"), pre)

	// Stub lstat/remove to simulate existing symlink detection and removal
	oldLs := goLstat
	oldRm := goRemove
	oldSym := goSymlink
	calledRemove := 0
	goLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	goRemove = func(path string) error { calledRemove++; return os.Remove(path) }
	goSymlink = os.Symlink
	assert.NoError(t, p.createSymlink("pkg:golang/github.com/acme/tool"))
	assert.GreaterOrEqual(t, calledRemove, 1)

	// Now force symlink creation error
	goSymlink = func(string, string) error { return errors.New("sym") }
	err := p.createSymlink("pkg:golang/github.com/acme/tool")
	assert.Error(t, err)

	goLstat, goRemove, goSymlink = oldLs, oldRm, oldSym
}

func TestGolangRemoveBin_ErrorOnRemove(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// registry with bin
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "v1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/acme/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.GetData(true)

	// create binary file so goStat finds it
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	binPath := filepath.Join(gobin, "tool")
	assert.NoError(t, os.WriteFile(binPath, []byte(""), 0755))

	oldRm := goRemove
	goRemove = func(string) error { return errors.New("rm") }
	defer func() { goRemove = oldRm }()

	assert.Error(t, p.removeBin("pkg:golang/github.com/acme/tool"))
}

func TestGolangClean_BinaryRemoveErrorAndLocalRemoveError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Add a desired package
	_ = lppGoAdd("pkg:golang/github.com/acme/tool", "v1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "v1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/acme/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.GetData(true)

	// create a binary so Clean tries to remove it and emit error
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(gobin, "tool"), []byte(""), 0755))

	oldRm := goRemove
	goRemove = func(string) error { return errors.New("rm") }
	// Clean should continue despite binary removal error (logs internally)
	assert.True(t, p.Clean())
	goRemove = oldRm

	// Now force local package removal to fail via injectable and expect Clean to return false
	oldLppRemove := lppGoRemove
	lppGoRemove = func(string) error { return errors.New("lppRemove") }
	defer func() { lppGoRemove = oldLppRemove }()
	// Re-add package so Clean iterates and hits lppGoRemove error
	_ = lppGoAdd("pkg:golang/github.com/acme/tool", "v1.0.0")
	assert.False(t, p.Clean())
}

func TestGolangSync_DirMkdirErrorAndModInitError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()

	// 1) Directory creation error
	oldStat := goStat
	oldMkdir := goMkdir
	goStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	goMkdir = func(string, os.FileMode) error { return errors.New("mkdir") }
	assert.False(t, p.Sync())
	goMkdir = oldMkdir
	goStat = oldStat

	// 2) go.mod init error
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// Add a package so packagesFound == true and we reach mod init path
	_ = lppGoAdd("pkg:golang/github.com/acme/tool", "v1.0.0")
	oldStat = goStat
	oldOut := goShellOut
	// Ensure go.mod is missing
	goStat = func(name string) (os.FileInfo, error) {
		if filepath.Base(name) == "go.mod" {
			return nil, os.ErrNotExist
		}
		return oldStat(name)
	}
	// Make go unavailable to force init path, but return error for init call specifically
	goShellOut = func(cmd string, args []string, dir string, env []string) (int, error) {
		if len(args) >= 2 && args[0] == "mod" && args[1] == "init" {
			return 1, errors.New("init")
		}
		return 0, nil
	}
	assert.False(t, p.Sync())
	goShellOut = oldOut
	goStat = oldStat
}

func TestGolangInstall_LatestFetchError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	oldCap := goShellOutCapture
	goShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	defer func() { goShellOutCapture = oldCap }()

	assert.False(t, p.Install("pkg:golang/github.com/acme/tool", "latest"))
}

func TestGolangSync_InstallErrorSetsAllOkFalse(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// ensure go.mod exists to skip mod init
	assert.NoError(t, os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "go.mod"), []byte("module zana"), 0644))
	// add desired package
	_ = lppGoAdd("pkg:golang/github.com/acme/tool", "v1.0.0")
	// stub goShellOut: go available ok, install fails
	oldOut := goShellOut
	goShellOut = func(cmd string, args []string, dir string, env []string) (int, error) {
		if len(args) == 1 && args[0] == "version" {
			return 0, nil
		}
		if len(args) >= 1 && args[0] == "install" {
			return 1, errors.New("install")
		}
		return 0, nil
	}
	defer func() { goShellOut = oldOut }()

	assert.False(t, p.Sync())
}
