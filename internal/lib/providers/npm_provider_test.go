package providers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/mistweaverco/zana-client/internal/lib/local_packages_parser"
	"github.com/mistweaverco/zana-client/internal/lib/registry_parser"
	"github.com/stretchr/testify/assert"
)

func TestNPMErrorBranches(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// readPackageJSON error
	oldRF := npmReadFile
	npmReadFile = func(string) ([]byte, error) { return nil, errors.New("boom") }
	_, err := p.readPackageJSON(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "x"))
	assert.Error(t, err)
	npmReadFile = oldRF

	// hasPackageJSONChanged branches
	oldStat := npmStat
	// package.json missing
	npmStat = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	assert.True(t, p.hasPackageJSONChanged())
	// lock missing
	npmStat = func(name string) (os.FileInfo, error) {
		if filepath.Base(name) == "package.json" {
			return fileInfoNow(t), nil
		}
		return nil, os.ErrNotExist
	}
	assert.True(t, p.hasPackageJSONChanged())
	// pkgStat error
	npmStat = func(name string) (os.FileInfo, error) {
		if filepath.Base(name) == "package.json" {
			return nil, errors.New("err")
		}
		return fileInfoNow(t), nil
	}
	assert.True(t, p.hasPackageJSONChanged())
	// lockStat error
	npmStat = func(name string) (os.FileInfo, error) {
		if filepath.Base(name) == "package.json" {
			return fileInfoNow(t), nil
		}
		return nil, errors.New("err")
	}
	assert.True(t, p.hasPackageJSONChanged())
	npmStat = oldStat

	// tryNpmCi no lock
	oldStat2 := npmStat
	npmStat = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	assert.False(t, p.tryNpmCi())
	npmStat = oldStat2

	// createPackageSymlinks symlink error and chmod error branches
	// prepare a fake package.json
	nm := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "pkg")
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	_ = os.MkdirAll(nm, 0755)
	pkgJSON := `{"name":"pkg","version":"1.0.0","bin":{"tool":"./bin/tool.js"}}`
	assert.NoError(t, os.WriteFile(filepath.Join(nm, "package.json"), []byte(pkgJSON), 0644))

	// existing symlink removal error
	oldL := npmLstat
	oldRm := npmRemove
	oldSym := npmSymlink
	oldCh := npmChmod
	npmLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	npmRemove = func(string) error { return errors.New("rmerr") }
	npmSymlink = func(oldname, newname string) error { return errors.New("symerr") }
	_ = p.createPackageSymlinks("pkg")

	// symlink ok, chmod error
	npmSymlink = func(oldname, newname string) error { return nil }
	npmChmod = func(string, os.FileMode) error { return errors.New("chmoderr") }
	err = p.createPackageSymlinks("pkg")
	assert.NoError(t, err)
	npmLstat, npmRemove, npmSymlink, npmChmod = oldL, oldRm, oldSym, oldCh

	// removePackageSymlinks removal error
	oldL = npmLstat
	oldRm = npmRemove
	npmLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	npmRemove = func(string) error { return errors.New("rmerr") }
	assert.NoError(t, p.removePackageSymlinks("pkg"))
	npmLstat, npmRemove = oldL, oldRm

	// removeAllSymlinks readDir error
	oldRD := npmReadDir
	npmReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	assert.Error(t, p.removeAllSymlinks())
	npmReadDir = oldRD

	// Clean dir remove error -> false
	oldRA := npmRemoveAll
	npmRemoveAll = func(string) error { return errors.New("rmalldir") }
	assert.False(t, p.Clean())
	npmRemoveAll = oldRA
}

func fileInfoNow(t *testing.T) os.FileInfo {
	t.Helper()
	f := filepath.Join(t.TempDir(), "tmp")
	_ = os.WriteFile(f, []byte("x"), 0644)
	fi, _ := os.Stat(f)
	_ = os.Chtimes(f, time.Now(), time.Now())
	return fi
}

func TestPyPiErrorBranches(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()

	// readPackageInfo errors
	oldRD := pipReadDir
	pipReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("boom") }
	_, err := p.readPackageInfo("/does/not/matter")
	assert.Error(t, err)
	pipReadDir = func(path string) ([]os.DirEntry, error) { return []os.DirEntry{}, nil }
	_, err = p.readPackageInfo(t.TempDir())
	assert.Error(t, err) // no info dir
	// create info dir but no metadata
	base := t.TempDir()
	_ = os.MkdirAll(filepath.Join(base, "x.egg-info"), 0755)
	pipReadDir = oldRD
	_, err = p.readPackageInfo(base)
	assert.Error(t, err)

	// createWrappers with lstat/remove/chmod errors
	_ = withTempZanaHome(t)
	p = NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	_ = local_packages_parser.AddLocalPackage("pkg:pypi/black", "1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "black", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/black"},
		Bin: map[string]string{"black": "black"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	oldLs, oldRm, oldCh := pipLstat, pipRemove, pipChmod
	pipLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	pipRemove = func(string) error { return errors.New("rm") }
	pipChmod = func(string, os.FileMode) error { return errors.New("chmod") }
	assert.NoError(t, p.createWrappers())
	pipLstat, pipRemove, pipChmod = oldLs, oldRm, oldCh

	// findSitePackagesDir errors
	oldSt := pipStat
	oldRDir := pipReadDir
	pipStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	assert.Equal(t, "", p.findSitePackagesDir())
	pipStat = oldSt
	pipReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	assert.Equal(t, "", p.findSitePackagesDir())
	pipReadDir = oldRDir

	// removeAllWrappers/removePackageWrappers removal error
	oldLs = pipLstat
	oldRm = pipRemove
	pipLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	pipRemove = func(string) error { return errors.New("rm") }
	assert.NoError(t, p.removeAllWrappers())
	assert.NoError(t, p.removePackageWrappers("black"))
	pipLstat, pipRemove = oldLs, oldRm

	// getLatestVersion invalid format
	oldCap := pipShellOutCapture
	pipShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		return 0, "no parens", nil
	}
	_, err = p.getLatestVersion("black")
	assert.Error(t, err)
	pipShellOutCapture = oldCap
}

func TestGolangErrorBranches(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	// registry item with no bin
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "nobin", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/nobin"},
		Bin: map[string]string{},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	assert.Error(t, p.createSymlink("pkg:golang/nobin"))
	assert.Error(t, p.removeSymlink("pkg:golang/nobin"))
	assert.Error(t, p.removeBin("pkg:golang/nobin"))

	// missing binary
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	assert.Error(t, p.createSymlink("pkg:golang/tool"))

	// Sync go unavailable
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldGo := goShellOut
	goShellOut = func(string, []string, string, []string) (int, error) { return 1, errors.New("x") }
	assert.False(t, p.Sync())
	goShellOut = oldGo
}

func TestPushingRemainingBranchesTo100(t *testing.T) {
	_ = withTempZanaHome(t)

	// NPM: Sync fast path (lock newer, all installed, no changes) and needsUpdate path with ci success
	np := NewProviderNPM()
	_ = os.MkdirAll(np.APP_PACKAGES_DIR, 0755)
	_ = local_packages_parser.AddLocalPackage("pkg:npm/a", "1.0.0")
	// desired
	pkgPath := filepath.Join(np.APP_PACKAGES_DIR, "package.json")
	assert.NoError(t, os.WriteFile(pkgPath, []byte("{}"), 0644))
	// lock newer matching
	lock := filepath.Join(np.APP_PACKAGES_DIR, "package-lock.json")
	lockData := `{"dependencies":{"a":{"version":"1.0.0"}}}`
	assert.NoError(t, os.WriteFile(lock, []byte(lockData), 0644))
	now := time.Now()
	_ = os.Chtimes(lock, now.Add(1*time.Hour), now.Add(1*time.Hour))
	// create node_modules bin so createPackageSymlinks can run
	_ = os.MkdirAll(filepath.Join(np.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	nm := filepath.Join(np.APP_PACKAGES_DIR, "node_modules", "a")
	_ = os.MkdirAll(nm, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(nm, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644))
	// fast path
	assert.True(t, np.Sync())

	// needsUpdate: change desired version and simulate ci success
	_ = local_packages_parser.AddLocalPackage("pkg:npm/a", "2.0.0")
	oldOut := npmShellOut
	npmShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	assert.True(t, np.Sync())
	npmShellOut = oldOut

	// NPM: Remove returns false when local RemoveLocalPackage fails
	oldRemoveLocal := lppRemove
	lppRemove = func(string) error { return errors.New("x") }
	assert.False(t, np.Remove("pkg:npm/a"))
	lppRemove = oldRemoveLocal

	// PyPI: Clean removeAll error and then success
	pp := NewProviderPyPi()
	_ = os.MkdirAll(pp.APP_PACKAGES_DIR, 0755)
	oldRA := pipRemoveAll
	pipRemoveAll = func(string) error { return errors.New("x") }
	assert.False(t, pp.Clean())
	pipRemoveAll = oldRA

	// Golang: Remove returns false when local RemoveLocalPackage fails
	gp := NewProviderGolang()
	_ = os.MkdirAll(gp.APP_PACKAGES_DIR, 0755)
	oldRemoveLocal2 := lppGoRemove
	lppGoRemove = func(string) error { return errors.New("x") }
	assert.False(t, gp.Remove("pkg:golang/x"))
	lppGoRemove = oldRemoveLocal2

	// Cargo: Remove returns false when local RemoveLocalPackage fails
	cp := NewProviderCargo()
	_ = os.MkdirAll(cp.APP_PACKAGES_DIR, 0755)
	oldRemoveLocal3 := lppCargoRemove
	lppCargoRemove = func(string) error { return errors.New("x") }
	assert.False(t, cp.Remove("pkg:cargo/x"))
	lppCargoRemove = oldRemoveLocal3
}

func TestMoreBranchesNPM(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Install: latest version fetch fails -> false
	oldCap := npmShellOutCapture
	npmShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	assert.False(t, p.Install("pkg:npm/pkg", "latest"))
	npmShellOutCapture = oldCap

	// Install: add local package fails -> false
	oldAdd := lppAdd
	lppAdd = func(string, string) error { return errors.New("add") }
	assert.False(t, p.Install("pkg:npm/pkg", "1.0.0"))
	lppAdd = oldAdd

	// Update: repo empty -> false
	assert.False(t, p.Update("pkg:npm/"))

	// removeAllSymlinks success path
	_ = os.MkdirAll(files.GetAppBinPath(), 0755)
	assert.NoError(t, p.removeAllSymlinks())
}

func TestMoreBranchesPyPI(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Install: latest fetch fails
	oldCap := pipShellOutCapture
	pipShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	assert.False(t, p.Install("pkg:pypi/black", "latest"))
	pipShellOutCapture = oldCap

	// Install: add fails
	oldAdd := lppPyAdd
	lppPyAdd = func(string, string) error { return errors.New("add") }
	assert.False(t, p.Install("pkg:pypi/black", "1.0.0"))
	lppPyAdd = oldAdd

	// Update: latest fetch fails
	pipShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	assert.False(t, p.Update("pkg:pypi/black"))
	pipShellOutCapture = oldCap

	// Remove: removeBin error -> false
	// Make removeBin fail by making findPackageInfoDir return ""
	// Construct a source where findSitePackagesDir returns empty and thus findPackageInfoDir returns ""
	assert.False(t, p.Remove("pkg:pypi/black"))

	// createPythonWrapperForCommand write error by pointing to a directory
	dir := filepath.Join(p.APP_PACKAGES_DIR, "bin", "d")
	_ = os.MkdirAll(dir, 0755)
	assert.Error(t, p.createPythonWrapperForCommand("echo true", dir))

	// Additional injection-based coverage to push to 100%
	// Close warning path for requirements.txt
	oldClose := pipClose
	pipClose = func(f *os.File) error { return errors.New("close") }
	// Force found=true by adding a desired pypi package
	_ = lppPyAdd("pkg:pypi/extra", "1.0.0")
	_ = p.generateRequirementsTxt()
	pipClose = oldClose

	// removePackageWrappers uses injectables for lstat/remove
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "wrap", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/wrap"},
		Bin: map[string]string{"wrap": "wrap"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	_ = os.WriteFile(filepath.Join(files.GetAppBinPath(), "wrap"), []byte(""), 0755)
	oldLs := pipLstat
	oldRm := pipRemove
	pipLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	pipRemove = func(string) error { return errors.New("rm") }
	_ = p.removePackageWrappers("wrap")
	pipLstat, pipRemove = oldLs, oldRm

	// Constructor pip/pip3 detection branches
	oldHas := pipHasCommand
	pipHasCommand = func(cmd string, args []string, env []string) bool {
		if cmd == "pip" {
			return false
		}
		if cmd == "pip3" {
			return true
		}
		return false
	}
	p2 := NewProviderPyPi()
	assert.NotNil(t, p2)
	// pip and pip3 both missing -> error log path (no assertion on log content)
	pipHasCommand = func(cmd string, args []string, env []string) bool { return false }
	_ = NewProviderPyPi()
	pipHasCommand = oldHas
}

func TestPyPiGetRepoEmptyAndCreateWrappersEarlyReturn(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	assert.Equal(t, "", p.getRepo("pkg:pypi"))
	// No desired packages -> createWrappers should return nil quickly
	assert.NoError(t, p.createWrappers())
}

func TestPyPiGenerateRequirements_WriteErrorAndCloseWarning(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// Add a desired PyPI package so found becomes true
	_ = lppPyAdd("pkg:pypi/pkg", "1.0.0")
	// Return a closed file from pipCreate so WriteString errors
	oldCreate := pipCreate
	oldClose := pipClose
	pipCreate = func(path string) (*os.File, error) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
		return f, nil
	}
	// Also trigger close warning for completeness
	pipClose = func(f *os.File) error { return errors.New("close") }
	assert.False(t, p.generateRequirementsTxt())
	// restore
	pipClose = oldClose
	pipCreate = oldCreate
}

func TestPyPiCreateWrappers_WrapperCreateErrorContinues(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired package with empty bin command to force wrapper creation error
	_ = lppPyAdd("pkg:pypi/wbad", "1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "wbad", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/wbad"},
		Bin: map[string]string{"bad": ""},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// Should not error overall
	assert.NoError(t, p.createWrappers())
}

func TestPyPiFindPackageInfoDir_ErrorAndContinues(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	// create lib/pythonX/site-packages
	lib := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages")
	_ = os.MkdirAll(lib, 0755)
	// Cause ReadDir on site-packages to error -> return ""
	oldRD := pipReadDir
	pipReadDir = func(path string) ([]os.DirEntry, error) {
		if path == lib {
			return nil, errors.New("boom")
		}
		return oldRD(path)
	}
	assert.Equal(t, "", p.findPackageInfoDir("pkg"))
	// Restore
	pipReadDir = oldRD

	// Now create entries that trigger both continue branches and still return ""
	// 1) A non-directory file
	_ = os.WriteFile(filepath.Join(lib, "file.txt"), []byte(""), 0644)
	// 2) A directory that is not *.dist-info or *.egg-info
	_ = os.MkdirAll(filepath.Join(lib, "notinfo"), 0755)
	assert.Equal(t, "", p.findPackageInfoDir("pkg"))
}

func TestPyPiParseEntryPoints_ErrorReturnsNil(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	// Directory without entry_points.txt -> pipReadFile will error
	tmp := filepath.Join(p.APP_PACKAGES_DIR, "tmpinfo")
	_ = os.MkdirAll(tmp, 0755)
	entries := p.parseEntryPointsFromInfoDir(tmp)
	assert.Nil(t, entries)
}

func TestPyPiSync_CreateDirErrorAndSkipInstalledInLoopAndFreezeError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	// 1) mkdir error
	oldStat := pipStat
	oldMkdir := pipMkdir
	pipStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	pipMkdir = func(string, os.FileMode) error { return errors.New("mkdir") }
	assert.False(t, p.Sync())
	// restore
	pipStat = oldStat
	pipMkdir = oldMkdir

	// 2) skip in-loop for installed + increment skippedCount + freeze error log + install failure allOk=false
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired c1 (installed) and c2 (to install)
	_ = lppPyAdd("pkg:pypi/c1", "1.0.0")
	_ = lppPyAdd("pkg:pypi/c2", "2.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "c1", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/c1"},
		Bin: map[string]string{"c1": "c1"},
	}, {
		Name: "c2", Version: "2.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/c2"},
		Bin: map[string]string{},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// requirements
	assert.True(t, p.generateRequirementsTxt())
	// freeze returns c1 only; then later return non-zero to exercise error path too
	oldCap := pipShellOutCapture
	called := 0
	pipShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		// only used for freeze in this test
		called++
		if called == 1 {
			return 0, "c1==1.0.0", nil
		}
		return 1, "", errors.New("freeze")
	}
	// install fails for c2
	oldOut := pipShellOut
	pipShellOut = func(string, []string, string, []string) (int, error) { return 1, errors.New("install") }
	assert.False(t, p.Sync())
	// restore
	pipShellOut = oldOut
	pipShellOutCapture = oldCap
}

func TestPyPiFindSitePackages_ReadDirErrorAndEmpty(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	// Create lib dir to pass first stat
	lib := filepath.Join(p.APP_PACKAGES_DIR, "lib")
	_ = os.MkdirAll(lib, 0755)
	// readDir error -> returns ""
	oldRD := pipReadDir
	pipReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	assert.Equal(t, "", p.findSitePackagesDir())
	// restore and create empty python dir without site-packages stat success -> still ""
	pipReadDir = oldRD
	py := filepath.Join(lib, "python3.11")
	_ = os.MkdirAll(py, 0755)
	assert.Equal(t, "", p.findSitePackagesDir())
}

func TestPyPiSync_SkipBranchInLoop(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired c1 installed, c2 needs install
	_ = lppPyAdd("pkg:pypi/c1", "1.0.0")
	_ = lppPyAdd("pkg:pypi/c2", "2.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "c1", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/c1"},
		Bin: map[string]string{},
	}, {
		Name: "c2", Version: "2.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/c2"},
		Bin: map[string]string{},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// ensure requirements created
	_ = p.generateRequirementsTxt()
	// Freeze returns c1 installed for both areAllPackagesInstalled and the installed map
	oldCap := pipShellOutCapture
	pipShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "c1==1.0.0", nil }
	// Install succeeds for c2 so Sync returns true
	oldOut := pipShellOut
	pipShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	assert.True(t, p.Sync())
	pipShellOut = oldOut
	pipShellOutCapture = oldCap
}

func TestPyPiGetLatestVersion_NoVersions(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	oldCap := pipShellOutCapture
	pipShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "()", nil }
	_, err := p.getLatestVersion("pkg")
	assert.Error(t, err)
	pipShellOutCapture = oldCap
}

func TestPyPiRemove_WrapperRemovalErrorAndLocalRemoveError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// Make removePackageWrappers return an error via lstat/remove injectables
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// Ensure bin removal succeeds so we reach wrappers and local remove branches
	lib := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages")
	_ = os.MkdirAll(lib, 0755)
	info := filepath.Join(lib, "tool-1.0.0.dist-info")
	_ = os.MkdirAll(info, 0755)
	_ = os.WriteFile(filepath.Join(info, "entry_points.txt"), []byte("[console_scripts]\ntool=t:m\n"), 0644)
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "bin", "tool"), []byte(""), 0755)
	// Cause wrapper removal to try and warn by making file exist then removal fail
	_ = os.WriteFile(filepath.Join(files.GetAppBinPath(), "tool"), []byte(""), 0755)
	oldLs := pipLstat
	oldRm := pipRemove
	pipLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	pipRemove = func(string) error { return errors.New("rm") }
	// Now make local package removal fail to hit the log and false return
	oldLocalRemove := lppPyRemove
	lppPyRemove = func(string) error { return errors.New("local-remove") }
	assert.False(t, p.Remove("pkg:pypi/tool"))
	// restore
	lppPyRemove = oldLocalRemove
	pipLstat, pipRemove = oldLs, oldRm
}

func TestPyPiRemove_LocalRemoveErrorReturnsFalse(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// Registry and files so removeBin succeeds
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	lib := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages")
	_ = os.MkdirAll(lib, 0755)
	info := filepath.Join(lib, "tool-1.0.0.dist-info")
	_ = os.MkdirAll(info, 0755)
	_ = os.WriteFile(filepath.Join(info, "entry_points.txt"), []byte("[console_scripts]\ntool=t:m\n"), 0644)
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "bin", "tool"), []byte(""), 0755)
	// Make local remove fail so Remove returns false at the targeted line
	oldLocalRemove := lppPyRemove
	lppPyRemove = func(string) error { return errors.New("local-remove") }
	assert.False(t, p.Remove("pkg:pypi/tool"))
	lppPyRemove = oldLocalRemove
}

func TestPyPiRemove_InvalidSourceAndRemoveBinFailure(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// invalid source id for removeBin
	assert.Error(t, p.removeBin("pkg:pypi/"))
	// setup info dir and bin then make removal fail
	lib := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages")
	_ = os.MkdirAll(lib, 0755)
	info := filepath.Join(lib, "tool-1.0.0.dist-info")
	_ = os.MkdirAll(info, 0755)
	_ = os.WriteFile(filepath.Join(info, "entry_points.txt"), []byte("[console_scripts]\ntool=t:m\n"), 0644)
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "bin", "tool"), []byte(""), 0755)
	oldLs := pipLstat
	oldRm := pipRemove
	pipLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	pipRemove = func(string) error { return errors.New("rm") }
	assert.Error(t, p.removeBin("pkg:pypi/tool"))
	pipLstat, pipRemove = oldLs, oldRm
}

func TestMoreBranchesGolang(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Install: add fails -> false
	oldAdd := lppGoAdd
	lppGoAdd = func(string, string) error { return errors.New("add") }
	assert.False(t, p.Install("pkg:golang/mod", "1.0.0"))
	lppGoAdd = oldAdd

	// Update: latest fetch fails -> false
	oldCap := goShellOutCapture
	goShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	assert.False(t, p.Update("pkg:golang/mod"))
	goShellOutCapture = oldCap

	// Clean loop path: setup a package and registry with bin; induce remove symlink error
	_ = local_packages_parser.AddLocalPackage("pkg:golang/github.com/acme/tool", "1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/acme/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	oldLs := goLstat
	oldRm := goRemove
	goLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	goRemove = func(string) error { return errors.New("rm") }
	assert.True(t, p.Clean())
	goLstat, goRemove = oldLs, oldRm

	// Sync: go mod init path and install error path
	gm := filepath.Join(p.APP_PACKAGES_DIR, "go.mod")
	_ = os.Remove(gm)
	oldOut := goShellOut
	goShellOut = func(cmd string, args []string, dir string, env []string) (int, error) {
		if len(args) >= 2 && args[0] == "mod" && args[1] == "init" {
			return 0, nil
		}
		return 1, errors.New("install")
	}
	// Add desired package
	_ = local_packages_parser.AddLocalPackage("pkg:golang/github.com/a/b", "1.0.0")
	assert.False(t, p.Sync())
	goShellOut = oldOut
}

func TestMoreBranchesCargo(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Update: latest fetch error
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	assert.False(t, p.Update("pkg:cargo/x"))
	cargoShellOutCapture = oldCap

	// getInstalledCrates error path
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	m := p.getInstalledCrates()
	assert.Len(t, m, 0)
	cargoShellOutCapture = oldCap
}

func TestNPMNeedsUpdateCiFailThenInstallIndividually(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired v2.0.0
	_ = lppAdd("pkg:npm/a", "2.0.0")
	// lock newer with v1.0.0
	pkgPath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	_ = os.WriteFile(pkgPath, []byte("{}"), 0644)
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"}}}`), 0644)
	now := time.Now()
	_ = os.Chtimes(lock, now.Add(1*time.Hour), now.Add(1*time.Hour))
	// prepare node_modules/a package.json and .bin target
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	nm := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	_ = os.MkdirAll(nm, 0755)
	_ = os.WriteFile(filepath.Join(nm, "package.json"), []byte(`{"name":"a","version":"2.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	// tryNpmCi should fail
	oldOut := npmShellOut
	npmShellOut = func(cmd string, args []string, dir string, env []string) (int, error) {
		if len(args) == 1 && args[0] == "ci" {
			return 1, errors.New("ci")
		}
		// install individually succeeds
		return 0, nil
	}
	// lock removal error should be tolerated
	oldRm := npmRemove
	npmRemove = func(string) error { return errors.New("rm") }
	assert.True(t, p.Sync())
	npmRemove = oldRm
	npmShellOut = oldOut
}

func TestPyPiAreAllInstalledTrueTriggersWrappers(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired black==1.0.0
	_ = lppPyAdd("pkg:pypi/black", "1.0.0")
	// registry with wrapper bin
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "black", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/black"},
		Bin: map[string]string{"black": "black"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// freeze shows installed equal
	oldCap := pipShellOutCapture
	pipShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "black==1.0.0", nil }
	// ensure wrappers can be created
	oldCh := pipChmod
	pipChmod = func(string, os.FileMode) error { return nil }
	assert.True(t, p.Sync())
	pipChmod = oldCh
	pipShellOutCapture = oldCap
}

func TestGolangSyncInstallSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired package
	_ = lppGoAdd("pkg:golang/github.com/x/y", "1.0.0")
	// registry with bin
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "y", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/x/y"},
		Bin: map[string]string{"y": "y"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// ensure go.mod missing -> mod init path
	_ = os.Remove(filepath.Join(p.APP_PACKAGES_DIR, "go.mod"))
	// pre-create installed binary so createSymlink can find it
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	_ = os.WriteFile(filepath.Join(gobin, "y"), []byte(""), 0755)
	// go commands succeed
	oldOut := goShellOut
	goShellOut = func(cmd string, args []string, dir string, env []string) (int, error) { return 0, nil }
	assert.True(t, p.Sync())
	goShellOut = oldOut
}

func TestCargoLatestResolutionAndUninstallFail(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired with latest
	_ = lppCargoAdd("pkg:cargo/cr", "latest")
	// search returns version
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		if len(args) > 0 && args[0] == "search" {
			return 0, "cr = \"1.2.3\"", nil
		}
		if len(args) > 0 && args[0] == "install" {
			return 0, "", nil
		}
		return 0, "", nil
	}
	// uninstall fails logs but proceed
	oldOut := cargoShellOut
	cargoShellOut = func(cmd string, args []string, dir string, env []string) (int, error) {
		if len(args) > 0 && args[0] == "uninstall" {
			return 1, errors.New("uninstall")
		}
		return 0, nil
	}
	// create cargo bin dir
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	assert.True(t, p.Sync())
	assert.True(t, p.Remove("pkg:cargo/cr"))
	cargoShellOut = oldOut
	cargoShellOutCapture = oldCap
}

// fake entry to simulate readDir scenarios
type fakeEntry struct {
	name string
	dir  bool
}

func (f fakeEntry) Name() string               { return f.name }
func (f fakeEntry) IsDir() bool                { return f.dir }
func (f fakeEntry) Type() os.FileMode          { return 0 }
func (f fakeEntry) Info() (os.FileInfo, error) { return fileInfoNow(nil), nil }

func TestMoreEdgesNpmAndPypiAndCargo(t *testing.T) {
	_ = withTempZanaHome(t)
	// NPM: isPackageInstalled true triggers skip path with symlink creation
	np := NewProviderNPM()
	_ = os.MkdirAll(np.APP_PACKAGES_DIR, 0755)
	_ = lppAdd("pkg:npm/a", "1.0.0")
	// create node_modules/a with version 1.0.0 and .bin target
	_ = os.MkdirAll(filepath.Join(np.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(np.APP_PACKAGES_DIR, "node_modules", "a")
	_ = os.MkdirAll(an, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	assert.True(t, np.Sync())

	// NPM: removeAllSymlinks skips directories
	_ = os.MkdirAll(filepath.Join(files.GetAppBinPath(), "subdir"), 0755)
	assert.NoError(t, np.removeAllSymlinks())

	// PyPI: Sync returns early when no packages
	pp := NewProviderPyPi()
	_ = os.MkdirAll(pp.APP_PACKAGES_DIR, 0755)
	assert.True(t, pp.Sync())

	// Cargo: removeAllSymlinks handles Lstat err via fake readDir; and Remove error when target under cargo
	cp := NewProviderCargo()
	_ = os.MkdirAll(cp.APP_PACKAGES_DIR, 0755)
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{fakeEntry{name: "ghost", dir: false}}, nil }
	oldLs := cargoLstat
	oldRl := cargoReadlink
	oldRm := cargoRemove
	cargoLstat = func(string) (os.FileInfo, error) { return nil, errors.New("lstat") }
	assert.NoError(t, cp.removeAllSymlinks())
	// simulate symlink pointing into cargo and remove failing
	cargoReadDir = oldRD
	// create cargo bin and a symlink to it under zana bin
	cbin := filepath.Join(cp.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cbin, 0755)
	zbin := files.GetAppBinPath()
	sl := filepath.Join(zbin, "sym")
	_ = os.Symlink(filepath.Join(cbin, "sym"), sl)
	cargoLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	cargoReadlink = func(string) (string, error) { return filepath.Join(cbin, "sym"), nil }
	cargoRemove = func(string) error { return errors.New("rm") }
	assert.NoError(t, cp.removeAllSymlinks())
	cargoLstat, cargoReadlink, cargoRemove = oldLs, oldRl, oldRm
}
func TestCargoErrorBranches(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// createSymlinks removal/chmod error
	cargoBin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cargoBin, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(cargoBin, "mybin"), []byte(""), 0755))
	oldLs, oldRm, oldCh, oldSym := cargoLstat, cargoRemove, cargoChmod, cargoSymlink
	cargoLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	cargoRemove = func(string) error { return errors.New("rm") }
	cargoChmod = func(string, os.FileMode) error { return errors.New("chmod") }
	cargoSymlink = func(string, string) error { return errors.New("sym") }
	assert.NoError(t, os.MkdirAll(files.GetAppBinPath(), 0755))
	assert.NoError(t, p.createSymlinks())
	cargoLstat, cargoRemove, cargoChmod, cargoSymlink = oldLs, oldRm, oldCh, oldSym

	// removeAllSymlinks over real symlink (already covered by other tests), add readlink error by pointing at non-cargo
	zbin := files.GetAppBinPath()
	sl := filepath.Join(zbin, "notcargo")
	_ = os.Symlink(filepath.Join(t.TempDir(), "x"), sl)
	assert.NoError(t, p.removeAllSymlinks())
}

func TestCargoSync_CreateDirErrorAndUnavailable(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	// mkdir error
	oldStat, oldMkdir := cargoStat, cargoMkdir
	cargoStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	cargoMkdir = func(string, os.FileMode) error { return errors.New("mkdir") }
	assert.False(t, p.Sync())
	cargoStat, cargoMkdir = oldStat, oldMkdir

	// cargo unavailable
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return false }
	assert.False(t, p.Sync())
	cargoHasCommand = oldHas
}

func TestCargoSync_InstallErrorSetsAllOkFalse(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired crate not installed
	_ = lppCargoAdd("pkg:cargo/tool", "1.2.3")
	// installed empty map: make ShellOutCapture return no list
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "", nil }
	// install fails
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 1, errors.New("install") }
	assert.False(t, p.Sync())
	cargoShellOut = oldOut
	cargoShellOutCapture = oldCap
}

func TestCargoClean_RemoveAllErrorReturnsFalse(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// make removeAll fail
	oldRemoveAll := cargoRemoveAll
	cargoRemoveAll = func(string) error { return errors.New("rmall") }
	assert.False(t, p.Clean())
	cargoRemoveAll = oldRemoveAll
}

func TestCargoCreateSymlinks_SkipDirAndRemovalSuccessAndSymlinkSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	// prepare cargo bin with a directory and a file
	cbin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(filepath.Join(cbin, "subdir"), 0755)
	_ = os.MkdirAll(cbin, 0755)
	// create a dummy binary file
	bin := filepath.Join(cbin, "mybin")
	_ = os.WriteFile(bin, []byte(""), 0755)
	// ensure zana bin has an existing symlink to remove successfully
	zbin := files.GetAppBinPath()
	_ = os.MkdirAll(zbin, 0755)
	sl := filepath.Join(zbin, "mybin")
	_ = os.Symlink(bin, sl)
	// make removal succeed and chmod succeed
	oldRm, oldCh := cargoRemove, cargoChmod
	cargoRemove = func(string) error { return nil }
	cargoChmod = func(string, os.FileMode) error { return nil }
	assert.NoError(t, p.createSymlinks())
	cargoRemove, cargoChmod = oldRm, oldCh
}

func TestCargoCreateSymlinks_ReadDirError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	cbin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cbin, 0755)
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	assert.Error(t, p.createSymlinks())
	cargoReadDir = oldRD
}

func TestCargoCreateSymlinks_ChmodErrorAfterSymlinkSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	cbin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cbin, 0755)
	// create a binary
	bin := filepath.Join(cbin, "b1")
	_ = os.WriteFile(bin, []byte(""), 0755)
	// ensure no pre-existing symlink
	_ = os.MkdirAll(files.GetAppBinPath(), 0755)
	oldSym := cargoSymlink
	oldCh := cargoChmod
	cargoSymlink = func(string, string) error { return nil }
	cargoChmod = func(string, os.FileMode) error { return errors.New("chmod") }
	assert.NoError(t, p.createSymlinks())
	cargoChmod = oldCh
	cargoSymlink = oldSym
}

func TestCargoCreateSymlinks_RemoveExistingSymlinkWarning(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	cbin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cbin, 0755)
	// simulate cargo bin entries via injectable readDir
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{fakeEntry{name: "warnbin", dir: false}}, nil }
	// zana bin existing symlink to trigger removal warning
	zbin := files.GetAppBinPath()
	_ = os.MkdirAll(zbin, 0755)
	sl := filepath.Join(zbin, "warnbin")
	_ = os.Symlink(filepath.Join(cbin, "warnbin"), sl)
	oldLs := cargoLstat
	oldRm := cargoRemove
	oldSym := cargoSymlink
	oldCh := cargoChmod
	cargoLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	cargoRemove = func(string) error { return errors.New("rm") }
	cargoSymlink = func(string, string) error { return nil }
	cargoChmod = func(string, os.FileMode) error { return nil }
	assert.NoError(t, p.createSymlinks())
	cargoReadDir = oldRD
	cargoChmod = oldCh
	cargoSymlink = oldSym
	cargoRemove = oldRm
	cargoLstat = oldLs
}

func TestCargoCreateSymlinks_NoBinDirReturnsNil(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	// Do not create bin dir
	assert.NoError(t, p.createSymlinks())
}

func TestCargoRemoveAllSymlinks_NonSymlinkAndReadlinkError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// create a regular file in zana bin (non-symlink)
	zbin := files.GetAppBinPath()
	_ = os.MkdirAll(zbin, 0755)
	f := filepath.Join(zbin, "regular")
	_ = os.WriteFile(f, []byte(""), 0644)
	// create a symlink that will cause readlink error
	sl := filepath.Join(zbin, "badlink")
	_ = os.Symlink(filepath.Join(p.APP_PACKAGES_DIR, "bin", "missing"), sl)
	oldRl := cargoReadlink
	cargoReadlink = func(string) (string, error) { return "", errors.New("readlink") }
	assert.NoError(t, p.removeAllSymlinks())
	cargoReadlink = oldRl
}

func TestCargoRemoveAllSymlinks_RemoveWarning(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	// prepare cargo bin and zana bin symlink pointing into cargo bin
	cbin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cbin, 0755)
	zbin := files.GetAppBinPath()
	_ = os.MkdirAll(zbin, 0755)
	sl := filepath.Join(zbin, "s1")
	_ = os.Symlink(filepath.Join(cbin, "s1"), sl)
	oldLs := cargoLstat
	oldRl := cargoReadlink
	oldRm := cargoRemove
	// Use real Lstat so ModeSymlink is detected on the actual symlink we created
	cargoLstat = os.Lstat
	cargoReadlink = func(string) (string, error) { return filepath.Join(cbin, "s1"), nil }
	cargoRemove = func(string) error { return errors.New("rm") }
	assert.NoError(t, p.removeAllSymlinks())
	cargoRemove = oldRm
	cargoReadlink = oldRl
	cargoLstat = oldLs
}

func TestCargoSync_SkipInstalledWithLockUpdateWarningAndCreateSymlinksError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired latest
	_ = lppCargoAdd("pkg:cargo/tool", "latest")
	// installed shows same version 1.2.3 and also ensure getInstalledCrates path is executed first
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		if len(args) > 0 && args[0] == "install" && len(args) > 1 && args[1] == "--list" {
			return 0, "tool v1.2.3:", nil
		}
		if len(args) > 0 && args[0] == "search" {
			return 0, "tool = \"1.2.3\"", nil
		}
		return 0, "", nil
	}
	// updating lockfile fails to hit warning path (both when desired was latest and also after successful install)
	oldAdd := lppCargoAdd
	lppCargoAdd = func(string, string) error { return errors.New("update-lock") }
	// createSymlinks error after loop
	call := 0
	oldRD := cargoReadDir
	cargoReadDir = func(path string) ([]os.DirEntry, error) {
		call++
		if call == 1 {
			return []os.DirEntry{}, nil // for createSymlinks in removeAll path elsewhere, just safe
		}
		return nil, errors.New("readdir") // trigger error log in createSymlinks
	}
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Sync())
	// restore
	cargoHasCommand = oldHas
	cargoReadDir = oldRD
	lppCargoAdd = oldAdd
	cargoShellOutCapture = oldCap
}

func TestCargoSync_LatestResolutionErrorSetsAllOkFalse(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	_ = lppCargoAdd("pkg:cargo/tool", "latest")
	// search fails
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("search") }
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	// no-op symlinks
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{}, nil }
	assert.False(t, p.Sync())
	cargoReadDir = oldRD
	cargoHasCommand = oldHas
	cargoShellOutCapture = oldCap
}

func TestCargoSync_CreateSymlinksErrorLoggedWhenNoDesired(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	// empty desired
	oldGet := lppCargoGetDataForProvider
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	// force createSymlinks to error via readDir
	oldRD := cargoReadDir
	// ensure cargo bin dir exists so createSymlinks does not short-circuit
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	cargoReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	assert.True(t, p.Sync())
	cargoReadDir = oldRD
	lppCargoGetDataForProvider = oldGet
	cargoHasCommand = oldHas
}

func TestCargoInstall_LatestResolveErrorReturnsFalse(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("search") }
	assert.False(t, p.Install("pkg:cargo/tool", "latest"))
	cargoShellOutCapture = oldCap
}

func TestCargoRemove_RemoveAllSymlinksErrorLogs(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	oldLocalRemove := lppCargoRemove
	lppCargoRemove = func(string) error { return nil }
	oldRD := cargoReadDir
	// ensure cargo bin dir exists so later createSymlinks attempts and errors
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	cargoReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	// ensure Sync afterwards returns true
	oldGet := lppCargoGetDataForProvider
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Remove("pkg:cargo/tool"))
	cargoHasCommand = oldHas
	lppCargoGetDataForProvider = oldGet
	cargoReadDir = oldRD
	lppCargoRemove = oldLocalRemove
	cargoShellOut = oldOut
}

func TestCargoSync_InstallLatestSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	_ = lppCargoAdd("pkg:cargo/tool", "latest")
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		if len(args) > 1 && args[0] == "install" && args[1] == "--list" {
			return 0, "", nil
		}
		if len(args) > 0 && args[0] == "search" {
			return 0, "tool = \"1.2.3\"", nil
		}
		return 0, "", nil
	}
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	// make createSymlinks no-op
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{}, nil }
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Sync())
	cargoHasCommand = oldHas
	cargoReadDir = oldRD
	cargoShellOut = oldOut
	cargoShellOutCapture = oldCap
}

func TestCargoSync_InstallLatestLockUpdateWarning(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired latest so pkg.Version != desiredVersion after resolution
	_ = lppCargoAdd("pkg:cargo/tool", "latest")
	// resolve latest and show not installed
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		if len(args) > 1 && args[0] == "install" && args[1] == "--list" {
			return 0, "", nil
		}
		if len(args) > 0 && args[0] == "search" {
			return 0, "tool = \"1.2.3\"", nil
		}
		return 0, "", nil
	}
	// install succeeds
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	// cause post-install lock update to warn
	oldAdd := lppCargoAdd
	lppCargoAdd = func(sourceId, version string) error { return errors.New("update-lock") }
	// make symlink creation no-op
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{}, nil }
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Sync())
	// restore
	cargoHasCommand = oldHas
	cargoReadDir = oldRD
	lppCargoAdd = oldAdd
	cargoShellOut = oldOut
	cargoShellOutCapture = oldCap
}

func TestCargoInstall_SpecificVersionSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldAdd := lppCargoAdd
	lppCargoAdd = func(string, string) error { return nil }
	oldGet := lppCargoGetDataForProvider
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Install("pkg:cargo/tool", "1.2.3"))
	cargoHasCommand = oldHas
	lppCargoGetDataForProvider = oldGet
	lppCargoAdd = oldAdd
}

func TestCargoRemove_LocalRemoveError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	oldLocalRemove := lppCargoRemove
	lppCargoRemove = func(string) error { return errors.New("rm-local") }
	assert.False(t, p.Remove("pkg:cargo/tool"))
	lppCargoRemove = oldLocalRemove
	cargoShellOut = oldOut
}

func TestCargoInstall_InvalidAndLatestAddError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// invalid source id
	assert.False(t, p.Install("pkg:cargo/", "1.0.0"))
	// latest resolves but add fails
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "tool = \"1.2.3\"", nil }
	oldAdd := lppCargoAdd
	lppCargoAdd = func(string, string) error { return errors.New("add") }
	assert.False(t, p.Install("pkg:cargo/tool", "latest"))
	lppCargoAdd = oldAdd
	cargoShellOutCapture = oldCap
}

func TestCargoRemove_InvalidAndCreateSymlinksError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// invalid source id
	assert.False(t, p.Remove("pkg:cargo/"))
	// happy uninstall and local remove, but createSymlinks error path
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	oldLocalRemove := lppCargoRemove
	lppCargoRemove = func(string) error { return nil }
	// readDir stub: first call (removeAllSymlinks) returns empty, second (createSymlinks) errors
	count := 0
	oldRD := cargoReadDir
	cargoReadDir = func(path string) ([]os.DirEntry, error) {
		count++
		if count == 1 {
			return []os.DirEntry{}, nil
		}
		return nil, errors.New("readdir")
	}
	// ensure Sync returns quickly after Remove
	oldGet := lppCargoGetDataForProvider
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Remove("pkg:cargo/tool"))
	// restore
	cargoHasCommand = oldHas
	lppCargoGetDataForProvider = oldGet
	cargoReadDir = oldRD
	lppCargoRemove = oldLocalRemove
	cargoShellOut = oldOut
}

func TestCargoRemove_UninstallErrorProceeds(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 1, errors.New("uninstall") }
	oldLocalRemove := lppCargoRemove
	lppCargoRemove = func(string) error { return nil }
	// ensure removeAll and createSymlinks harmless
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{}, nil }
	// ensure Sync returns true
	oldGet := lppCargoGetDataForProvider
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Remove("pkg:cargo/tool"))
	cargoHasCommand = oldHas
	lppCargoGetDataForProvider = oldGet
	cargoReadDir = oldRD
	lppCargoRemove = oldLocalRemove
	cargoShellOut = oldOut
}

func TestCargoSync_SkipInvalidRepoEntryAndSpecificInstallSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// add invalid and valid entries
	_ = lppCargoAdd("pkg:cargo/", "1.0.0")
	_ = lppCargoAdd("pkg:cargo/tool", "1.2.3")
	// installed shows old version; force go through install
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		if len(args) > 1 && args[0] == "install" && args[1] == "--list" {
			return 0, "tool v1.0.0:", nil
		}
		return 0, "", nil
	}
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	// no-op symlink creation
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{}, nil }
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	assert.True(t, p.Sync())
	cargoHasCommand = oldHas
	cargoReadDir = oldRD
	cargoShellOut = oldOut
	cargoShellOutCapture = oldCap
}

func TestCargoInstall_EmptyVersionResolvesLatestSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "tool = \"1.2.3\"", nil }
	oldAdd := lppCargoAdd
	lppCargoAdd = func(string, string) error { return nil }
	oldGet := lppCargoGetDataForProvider
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	oldHas := cargoHasCommand
	cargoHasCommand = func(string, []string, []string) bool { return true }
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return []os.DirEntry{}, nil }
	assert.True(t, p.Install("pkg:cargo/tool", ""))
	cargoReadDir = oldRD
	cargoHasCommand = oldHas
	lppCargoGetDataForProvider = oldGet
	lppCargoAdd = oldAdd
	cargoShellOutCapture = oldCap
}

func TestNPMAllConditionalsToggle(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Case 1: generatePackageJSON returns false -> Sync returns true
	oldGet := lppGetData
	lppGetData = func(bool) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	assert.True(t, p.Sync())
	lppGetData = oldGet

	// Setup desired package a@2.0.0, lock newer with 1.0.0 to trigger needsUpdate
	_ = lppAdd("pkg:npm/a", "2.0.0")
	pkgPath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	_ = os.WriteFile(pkgPath, []byte("{}"), 0644)
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"}}}`), 0644)
	now := time.Now()
	_ = os.Chtimes(lock, now.Add(1*time.Hour), now.Add(1*time.Hour))
	// Ensure bin dirs exist
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	_ = os.MkdirAll(an, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"2.0.0","bin":{"a":"./bin/a.js"}}`), 0644)

	// Case 2: needsUpdate with ci success -> returns true early
	oldOut := npmShellOut
	npmShellOut = func(cmd string, args []string, dir string, env []string) (int, error) {
		if len(args) == 1 && args[0] == "ci" {
			return 0, nil
		}
		return 0, nil
	}
	assert.True(t, p.Sync())
	npmShellOut = oldOut

	// Case 3: Installing individually path with install failure -> returns false
	// Remove lock to skip lockExists branches
	_ = os.Remove(lock)
	// Desired b@1.0.0 not installed
	_ = lppAdd("pkg:npm/b", "1.0.0")
	bn := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "b")
	_ = os.MkdirAll(bn, 0755)
	_ = os.WriteFile(filepath.Join(bn, "package.json"), []byte(`{"name":"b","version":"0.9.0","bin":{"b":"./bin/b.js"}}`), 0644)
	oldOut2 := npmShellOut
	npmShellOut = func(cmd string, args []string, dir string, env []string) (int, error) {
		return 1, errors.New("install")
	}
	assert.False(t, p.Sync())
	npmShellOut = oldOut2
}

func TestNPMGeneratePackageJSONSkipsNonNpmAndCloseErrorAndEncodeError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Add a non-npm package and an npm package; ensure skip happens and found==true
	_ = local_packages_parser.AddLocalPackage("pkg:pypi/black", "1.0.0")
	_ = local_packages_parser.AddLocalPackage("pkg:npm/pkg", "1.2.3")
	assert.True(t, p.generatePackageJSON())

	// Encode error path: replace file with a directory so encoder.Encode fails to write
	oldCreate := npmCreate
	npmCreate = func(path string) (*os.File, error) {
		// create a directory at the path to cause encode error when opening file descriptor invalid
		_ = os.MkdirAll(path, 0755)
		return nil, errors.New("open as dir")
	}
	assert.False(t, p.generatePackageJSON())
	npmCreate = oldCreate

	// Close error path via injectable close
	filePath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	f, err := os.Create(filePath)
	assert.NoError(t, err)
	_ = f.Close()
	oldClose := npmClose
	npmClose = func(*os.File) error { return errors.New("close") }
	// generate writes and then triggers close warning; still returns true since found
	assert.True(t, p.generatePackageJSON())
	npmClose = oldClose

	// Encode error path: return a closed file so encoder writes fail
	oldCreate2 := npmCreate
	npmCreate = func(path string) (*os.File, error) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		_ = f.Close()
		return f, nil
	}
	assert.False(t, p.generatePackageJSON())
	npmCreate = oldCreate2
}

func TestNPMRemoveAllSymlinksWarnOnRemove(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(files.GetAppBinPath(), 0755)
	// create a dummy entry in bin
	f := filepath.Join(files.GetAppBinPath(), "dummy")
	assert.NoError(t, os.WriteFile(f, []byte(""), 0644))
	oldLs, oldRm := npmLstat, npmRemove
	npmLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	npmRemove = func(string) error { return errors.New("rm") }
	assert.NoError(t, p.removeAllSymlinks())
	npmLstat, npmRemove = oldLs, oldRm
}

func TestNPMCleanLogsErrorOnRemoveSymlinks(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	oldRD, oldRA, oldStat, oldMkdir, oldGet := npmReadDir, npmRemoveAll, npmStat, npmMkdir, lppGetData
	// make removeAllSymlinks error
	npmReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	// allow directory remove to succeed
	npmRemoveAll = func(string) error { return nil }
	// Sync path: make directory creation succeed and no packages found
	npmStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	npmMkdir = func(string, os.FileMode) error { return nil }
	lppGetData = func(bool) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	assert.True(t, p.Clean())
	// restore
	npmReadDir, npmRemoveAll, npmStat, npmMkdir, lppGetData = oldRD, oldRA, oldStat, oldMkdir, oldGet
}

func TestNPMSyncCreateDirError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	oldStat, oldMkdir := npmStat, npmMkdir
	npmStat = func(string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	npmMkdir = func(string, os.FileMode) error { return errors.New("mkdir") }
	assert.False(t, p.Sync())
	npmStat, npmMkdir = oldStat, oldMkdir
}

func TestNPMFastPathLogsCreateSymlinkErrors(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired a@1.0.0
	_ = lppAdd("pkg:npm/a", "1.0.0")
	// package.json and newer lock with matching version
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "package.json"), []byte("{}"), 0644)
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"}}}`), 0644)
	_ = os.Chtimes(lock, time.Now().Add(1*time.Hour), time.Now().Add(1*time.Hour))
	// node_modules setup with bin
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	_ = os.MkdirAll(an, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	// force symlink creation to fail so the log path is taken
	oldSym := npmSymlink
	npmSymlink = func(string, string) error { return errors.New("sym") }
	assert.True(t, p.Sync())
	npmSymlink = oldSym
}

func TestNPMInstallAndRemovePermutations(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// getInstalledPackagesFromLock cases
	assert.Equal(t, 0, len(p.getInstalledPackagesFromLock(filepath.Join(p.APP_PACKAGES_DIR, "missing.json"))))
	bad := filepath.Join(p.APP_PACKAGES_DIR, "bad.json")
	_ = os.WriteFile(bad, []byte("not-json"), 0644)
	assert.Equal(t, 0, len(p.getInstalledPackagesFromLock(bad)))
	good := filepath.Join(p.APP_PACKAGES_DIR, "good.json")
	_ = os.WriteFile(good, []byte(`{"dependencies":{"x":{"version":"1.2.3"}}}`), 0644)
	mp := p.getInstalledPackagesFromLock(good)
	assert.Equal(t, "1.2.3", mp["x"])

	// isPackageInstalled false when dir missing
	assert.False(t, p.isPackageInstalled("none", "1.0.0"))
	// false when readPackageJSON fails
	dir := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "broken")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{"), 0644)
	assert.False(t, p.isPackageInstalled("broken", "1.0.0"))
	// true when versions match
	okd := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "ok")
	_ = os.MkdirAll(okd, 0755)
	_ = os.WriteFile(filepath.Join(okd, "package.json"), []byte(`{"name":"ok","version":"1.0.0"}`), 0644)
	assert.True(t, p.isPackageInstalled("ok", "1.0.0"))

	// hasPackageJSONChanged false when lock newer
	pkgPath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(pkgPath, []byte("{}"), 0644)
	_ = os.WriteFile(lock, []byte("{}"), 0644)
	now := time.Now()
	_ = os.Chtimes(lock, now.Add(2*time.Hour), now.Add(2*time.Hour))
	assert.False(t, p.hasPackageJSONChanged())

	// createPackageSymlinks chmod success
	// Setup a package with bin
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	pa := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "pkg")
	_ = os.MkdirAll(pa, 0755)
	_ = os.WriteFile(filepath.Join(pa, "package.json"), []byte(`{"name":"pkg","version":"1.0.0","bin":{"cli":"./bin/cli.js"}}`), 0644)
	oldCh := npmChmod
	npmChmod = func(string, os.FileMode) error { return nil }
	assert.NoError(t, p.createPackageSymlinks("pkg"))
	npmChmod = oldCh

	// removePackageSymlinks removal success
	oldLs := npmLstat
	oldRm := npmRemove
	npmLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	npmRemove = func(string) error { return nil }
	assert.NoError(t, p.removePackageSymlinks("pkg"))
	npmLstat, npmRemove = oldLs, oldRm

	// Install success returns true even if createPackageSymlinks fails afterwards
	oldGet := lppGetData
	lppGetData = func(bool) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	assert.True(t, p.Install("pkg:npm/pkg", "1.0.0"))
	lppGetData = oldGet

	// Remove success (lppRemove ok) with Sync returning true from empty desired
	assert.True(t, p.Remove("pkg:npm/pkg"))
}

func TestPyPiMorePermutations(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// generateRequirementsTxt false when no packages
	oldGet := lppPyGetData
	lppPyGetData = func(bool) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	assert.False(t, p.generateRequirementsTxt())
	lppPyGetData = oldGet

	// createWrappers: no bin case then multi-bin
	// No bin case
	_ = lppPyAdd("pkg:pypi/nobin", "1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "nobin", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/nobin"},
		Bin: map[string]string{},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	assert.NoError(t, p.createWrappers())
	// Multi-bin
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/tool"},
		Bin: map[string]string{"a": "a", "b": "b"},
	}})
	_ = lppPyAdd("pkg:pypi/tool", "1.0.0")
	assert.NoError(t, p.createWrappers())

	// findSitePackagesDir success
	lib := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages")
	_ = os.MkdirAll(lib, 0755)
	assert.Equal(t, lib, p.findSitePackagesDir())

	// findPackageInfoDir with dist-info and egg-info
	infoDist := filepath.Join(lib, "pkg-1.0.0.dist-info")
	_ = os.MkdirAll(infoDist, 0755)
	assert.Contains(t, p.findPackageInfoDir("pkg"), "dist-info")
	infoEgg := filepath.Join(lib, "egg.egg-info")
	_ = os.MkdirAll(infoEgg, 0755)
	assert.Contains(t, p.findPackageInfoDir("egg"), "egg-info")

	// Remove: local remove fails -> false
	oldRm := lppPyRemove
	lppPyRemove = func(string) error { return errors.New("rm") }
	assert.False(t, p.Remove("pkg:pypi/tool"))
	lppPyRemove = oldRm

	// Update invalid repo
	assert.False(t, p.Update("pkg:pypi/"))
}

func TestGolangMorePermutations(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// generatePackageJSON: false then true
	assert.False(t, p.generatePackageJSON())
	_ = local_packages_parser.AddLocalPackage("pkg:golang/github.com/x/y", "v1.0.0")
	assert.True(t, p.generatePackageJSON())

	// Sync skip path when installed
	_ = lppGoAdd("pkg:golang/github.com/x/skip", "v1.0.0")
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	_ = os.WriteFile(filepath.Join(gobin, "skip"), []byte(""), 0755)
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "skip", Version: "v1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/x/skip"},
		Bin: map[string]string{"skip": "skip"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	oldGo := goShellOut
	goShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	assert.True(t, p.Sync())
	goShellOut = oldGo

	// getLatestVersion invalid output
	oldCap := goShellOutCapture
	goShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "onlyone", nil }
	_, err := p.getLatestVersion("mod")
	assert.Error(t, err)
	goShellOutCapture = oldCap
}

func TestCargoMorePermutations(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// Clean success when desired empty and Sync runs
	assert.True(t, p.Clean())

	// getLatestVersion not found
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "no matches", nil }
	_, err := p.getLatestVersion("crate")
	assert.Error(t, err)
	cargoShellOutCapture = oldCap

	// Sync skip installed path
	_ = lppCargoAdd("pkg:cargo/myc", "1.0.0")
	cargoShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "myc v1.0.0: desc", nil }
	assert.True(t, p.Sync())
	cargoShellOutCapture = oldCap

	// Install add failure -> false
	oldAdd := lppCargoAdd
	lppCargoAdd = func(string, string) error { return errors.New("add") }
	assert.False(t, p.Install("pkg:cargo/myc", "1.0.0"))
	lppCargoAdd = oldAdd
}

func TestGetRepoAllProviders(t *testing.T) {
	_ = withTempZanaHome(t)

	np := NewProviderNPM()
	assert.Equal(t, "pkg", np.getRepo("pkg:npm/pkg"))
	assert.Equal(t, "", np.getRepo("invalid"))

	pp := NewProviderPyPi()
	assert.Equal(t, "black", pp.getRepo("pkg:pypi/black"))
	assert.Equal(t, "", pp.getRepo("pkg:pypi/"))

	gp := NewProviderGolang()
	assert.Equal(t, "github.com/x/y", gp.getRepo("pkg:golang/github.com/x/y"))
	assert.Equal(t, "", gp.getRepo("pkg:golang/"))
	// non-matching prefix (missing trailing slash) exercises the else-branch returning empty
	assert.Equal(t, "", gp.getRepo("pkg:golang"))

	cp := NewProviderCargo()
	assert.Equal(t, "crate", cp.getRepo("pkg:cargo/crate"))
	assert.Equal(t, "", cp.getRepo("pkg:cargo/"))
	assert.Equal(t, "", cp.getRepo("invalid"))
}

func TestNPMGeneratePackageJSONCreateErrorAndCleanHappy(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// generatePackageJSON create error
	oldCreate := npmCreate
	npmCreate = func(string) (*os.File, error) { return nil, errors.New("create") }
	assert.False(t, p.generatePackageJSON())
	npmCreate = oldCreate

	// Clean happy path: removeAll ok, remove dir ok, Sync returns true because no packages
	oldRmAll := npmRemoveAll
	oldStat := npmStat
	oldMkdir := npmMkdir
	oldGet := lppGetData
	npmRemoveAll = func(string) error { return nil }
	npmStat = func(name string) (os.FileInfo, error) { return nil, os.ErrNotExist }
	npmMkdir = func(string, os.FileMode) error { return nil }
	lppGetData = func(bool) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	assert.True(t, p.Clean())
	// restore
	npmRemoveAll = oldRmAll
	npmStat = oldStat
	npmMkdir = oldMkdir
	lppGetData = oldGet
}

func TestPyPiGenerateRequirementsCreateErrorAndRemoveWrappersNoBinAndRemoveBinSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// generateRequirementsTxt create error
	oldCreate := pipCreate
	pipCreate = func(string) (*os.File, error) { return nil, errors.New("create") }
	assert.False(t, p.generateRequirementsTxt())
	pipCreate = oldCreate

	// removePackageWrappers no-bin returns nil
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "nobin", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/nobin"},
		Bin: map[string]string{},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	assert.NoError(t, p.removePackageWrappers("nobin"))

	// removeBin success: create info dir and bin file; ensure it deletes
	binDir := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(binDir, 0755)
	binFile := filepath.Join(binDir, "tool")
	assert.NoError(t, os.WriteFile(binFile, []byte(""), 0755))
	lib := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages")
	_ = os.MkdirAll(lib, 0755)
	info := filepath.Join(lib, "tool-1.0.0.dist-info")
	_ = os.MkdirAll(info, 0755)
	// entry_points.txt with console_scripts
	assert.NoError(t, os.WriteFile(filepath.Join(info, "entry_points.txt"), []byte("[console_scripts]\ntool = t:m\n"), 0644))
	assert.NoError(t, p.removeBin("pkg:pypi/tool"))
	// file should be gone
	_, err := os.Lstat(binFile)
	assert.Error(t, err)
}

func TestGolangCreateSymlinkSuccessAndGeneratePackageJSONCreateErrorAndUpdateInvalid(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// createSymlink success
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(gobin, "tool"), []byte(""), 0755))
	assert.NoError(t, p.createSymlink("pkg:golang/tool"))
	// symlink exists in zana bin
	_, err := os.Lstat(filepath.Join(files.GetAppBinPath(), "tool"))
	assert.NoError(t, err)

	// generatePackageJSON create error
	oldCreate := goCreate
	goCreate = func(string) (*os.File, error) { return nil, errors.New("create") }
	assert.False(t, p.generatePackageJSON())
	goCreate = oldCreate

	// Update invalid repo
	assert.False(t, p.Update("pkg:golang/"))
}

func TestCargoRepoAndCreateSymlinksSuccessAndCleanHappyAndInvalidUpdateRemove(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// createSymlinks success
	cargoBin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(cargoBin, 0755)
	assert.NoError(t, os.WriteFile(filepath.Join(cargoBin, "mybin"), []byte(""), 0755))
	_ = os.MkdirAll(files.GetAppBinPath(), 0755)
	assert.NoError(t, p.createSymlinks())

	// Clean happy path (make Sync return true quickly)
	oldHas := cargoHasCommand
	oldGet := lppCargoGetDataForProvider
	cargoHasCommand = func(string, []string, []string) bool { return true }
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	assert.True(t, p.Clean())
	cargoHasCommand = oldHas
	lppCargoGetDataForProvider = oldGet

	// Invalid repo branches
	assert.False(t, p.Remove("pkg:cargo/"))
	assert.False(t, p.Update("pkg:cargo/"))
}

func TestCargoRemoveAllSymlinksReadDirErrorAndCreateSymlinksNoBinDir(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	// readDir error -> propagate
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	assert.Error(t, p.removeAllSymlinks())
	cargoReadDir = oldRD

	// createSymlinks when cargo bin dir missing -> nil
	assert.NoError(t, p.createSymlinks())
}

func TestCargoCleanWithRemoveAllSymlinksErrorStillProceeds(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// force removeAllSymlinks to error
	oldRD := cargoReadDir
	cargoReadDir = func(string) ([]os.DirEntry, error) { return nil, errors.New("readdir") }
	// ensure RemoveAll OK via injectable
	oldRemoveAll := cargoRemoveAll
	cargoRemoveAll = func(string) error { return nil }
	// ensure Sync returns true (cargo available and desired empty)
	oldHas := cargoHasCommand
	oldGet := lppCargoGetDataForProvider
	cargoHasCommand = func(string, []string, []string) bool { return true }
	lppCargoGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	assert.True(t, p.Clean())
	// restore
	cargoReadDir = oldRD
	cargoRemoveAll = oldRemoveAll
	cargoHasCommand = oldHas
	lppCargoGetDataForProvider = oldGet
}

func TestNPMFastPathMultiPackageSymlinkLoop(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)

	// desired a@1.0.0 and b@2.0.0
	_ = lppAdd("pkg:npm/a", "1.0.0")
	_ = lppAdd("pkg:npm/b", "2.0.0")
	// create package.json and a newer lock with both versions matching
	pkgPath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	_ = os.WriteFile(pkgPath, []byte("{}"), 0644)
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"},"b":{"version":"2.0.0"}}}`), 0644)
	now := time.Now()
	_ = os.Chtimes(lock, now.Add(1*time.Hour), now.Add(1*time.Hour))
	// node_modules setup with bins for both
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	bn := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "b")
	_ = os.MkdirAll(an, 0755)
	_ = os.MkdirAll(bn, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	_ = os.WriteFile(filepath.Join(bn, "package.json"), []byte(`{"name":"b","version":"2.0.0","bin":{"b":"./bin/b.js"}}`), 0644)
	assert.True(t, p.Sync())
}

func TestNPMFastPathMultiPackageSymlinkLoopWithSymlinkErrors(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	_ = lppAdd("pkg:npm/a", "1.0.0")
	_ = lppAdd("pkg:npm/b", "2.0.0")
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "package.json"), []byte("{}"), 0644)
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"},"b":{"version":"2.0.0"}}}`), 0644)
	_ = os.Chtimes(lock, time.Now().Add(1*time.Hour), time.Now().Add(1*time.Hour))
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	bn := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "b")
	_ = os.MkdirAll(an, 0755)
	_ = os.MkdirAll(bn, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	_ = os.WriteFile(filepath.Join(bn, "package.json"), []byte(`{"name":"b","version":"2.0.0","bin":{"b":"./bin/b.js"}}`), 0644)
	oldSym := npmSymlink
	npmSymlink = func(string, string) error { return errors.New("sym") }
	assert.True(t, p.Sync())
	npmSymlink = oldSym
}

func TestNPMFastPathMultiPackageSymlinkLoopSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	_ = lppAdd("pkg:npm/a", "1.0.0")
	_ = lppAdd("pkg:npm/b", "2.0.0")
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "package.json"), []byte("{}"), 0644)
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"},"b":{"version":"2.0.0"}}}`), 0644)
	_ = os.Chtimes(lock, time.Now().Add(1*time.Hour), time.Now().Add(1*time.Hour))
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	bn := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "b")
	_ = os.MkdirAll(an, 0755)
	_ = os.MkdirAll(bn, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	_ = os.WriteFile(filepath.Join(bn, "package.json"), []byte(`{"name":"b","version":"2.0.0","bin":{"b":"./bin/b.js"}}`), 0644)
	oldCh := npmChmod
	npmChmod = func(string, os.FileMode) error { return nil }
	assert.True(t, p.Sync())
	npmChmod = oldCh
}

func TestNPMFastPathSecondLoopAllInstalled(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired
	_ = lppAdd("pkg:npm/a", "1.0.0")
	_ = lppAdd("pkg:npm/b", "2.0.0")
	// actual files (their real modtimes won't matter due to npmStat stub)
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "package.json"), []byte("{}"), 0644)
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json"), []byte(`{"dependencies":{"a":{"version":"1.0.0"},"b":{"version":"2.0.0"}}}`), 0644)
	// setup node_modules with bins
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	bn := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "b")
	_ = os.MkdirAll(an, 0755)
	_ = os.MkdirAll(bn, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	_ = os.WriteFile(filepath.Join(bn, "package.json"), []byte(`{"name":"b","version":"2.0.0","bin":{"b":"./bin/b.js"}}`), 0644)

	// Create controlled fi for lock newer and pkg older; also ensure hasPackageJSONChanged returns false
	tmp := t.TempDir()
	fakeLock := filepath.Join(tmp, "lock")
	fakePkg := filepath.Join(tmp, "pkg")
	_ = os.WriteFile(fakeLock, []byte("x"), 0644)
	_ = os.WriteFile(fakePkg, []byte("x"), 0644)
	_ = os.Chtimes(fakeLock, time.Now().Add(2*time.Hour), time.Now().Add(2*time.Hour))
	_ = os.Chtimes(fakePkg, time.Now(), time.Now())
	oldStat := npmStat
	npmStat = func(name string) (os.FileInfo, error) {
		switch filepath.Base(name) {
		case "package-lock.json":
			fi, _ := os.Stat(fakeLock)
			return fi, nil
		case "package.json":
			fi, _ := os.Stat(fakePkg)
			return fi, nil
		default:
			return oldStat(name)
		}
	}
	oldCh := npmChmod
	npmChmod = func(string, os.FileMode) error { return nil }

	assert.True(t, p.Sync())
	npmChmod = oldCh
	npmStat = oldStat
}

func TestNPMFastPathAllInstalledCallsSymlinkForEachPackage(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired packages
	_ = lppAdd("pkg:npm/a", "1.0.0")
	_ = lppAdd("pkg:npm/b", "2.0.0")
	// package.json and lock newer matching
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "package.json"), []byte("{}"), 0644)
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"},"b":{"version":"2.0.0"}}}`), 0644)
	_ = os.Chtimes(lock, time.Now().Add(1*time.Hour), time.Now().Add(1*time.Hour))
	// node_modules with bin for both packages
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	bn := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "b")
	_ = os.MkdirAll(an, 0755)
	_ = os.MkdirAll(bn, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	_ = os.WriteFile(filepath.Join(bn, "package.json"), []byte(`{"name":"b","version":"2.0.0","bin":{"b":"./bin/b.js"}}`), 0644)

	// Capture which symlinks were attempted to verify loop over desired
	called := map[string]int{}
	oldSym := npmSymlink
	npmSym := func(oldname, newname string) error {
		base := filepath.Base(newname)
		called[base]++
		return nil
	}
	npmSymlink = npmSym
	oldCh := npmChmod
	npmChmod = func(string, os.FileMode) error { return nil }

	assert.True(t, p.Sync())
	// Expect both a and b symlink attempts
	assert.GreaterOrEqual(t, called["a"], 1)
	assert.GreaterOrEqual(t, called["b"], 1)

	npmChmod = oldCh
	npmSymlink = oldSym
}

func TestNPMFastPathAllInstalledMixedSymlinkSuccessAndError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired packages
	_ = lppAdd("pkg:npm/a", "1.0.0")
	_ = lppAdd("pkg:npm/b", "2.0.0")
	// package.json and lock (lock newer)
	pkgPath := filepath.Join(p.APP_PACKAGES_DIR, "package.json")
	lock := filepath.Join(p.APP_PACKAGES_DIR, "package-lock.json")
	_ = os.WriteFile(pkgPath, []byte("{}"), 0644)
	_ = os.WriteFile(lock, []byte(`{"dependencies":{"a":{"version":"1.0.0"},"b":{"version":"2.0.0"}}}`), 0644)
	now := time.Now()
	_ = os.Chtimes(pkgPath, now, now)
	_ = os.Chtimes(lock, now.Add(1*time.Hour), now.Add(1*time.Hour))
	// node_modules with valid a and invalid b to force error on b
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	bn := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "b")
	_ = os.MkdirAll(an, 0755)
	_ = os.MkdirAll(bn, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	_ = os.WriteFile(filepath.Join(bn, "package.json"), []byte("{"), 0644) // invalid JSON -> error path
	// ensure chmod does not fail for the success package
	oldCh := npmChmod
	npmChmod = func(string, os.FileMode) error { return nil }
	assert.True(t, p.Sync())
	npmChmod = oldCh
}

func TestNPMUpdateLatestFetchFail(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	oldCap := npmShellOutCapture
	npmShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 1, "", errors.New("err") }
	assert.False(t, p.Update("pkg:npm/x"))
	npmShellOutCapture = oldCap
}

func TestNPMSkipPathSymlinkError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	_ = lppAdd("pkg:npm/a", "1.0.0")
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	an := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "a")
	_ = os.MkdirAll(an, 0755)
	_ = os.WriteFile(filepath.Join(an, "package.json"), []byte(`{"name":"a","version":"1.0.0","bin":{"a":"./bin/a.js"}}`), 0644)
	// No lock file so it goes to individual install path; mark package installed so skip branch executes
	oldSym := npmSymlink
	npmSymlink = func(string, string) error { return errors.New("sym") }
	assert.True(t, p.Sync())
	npmSymlink = oldSym
}

func TestNPMInstallPostSyncSymlinkError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// Ensure Sync will return true quickly by faking no packages
	oldGetProv := lppGetDataForProvider
	lppGetDataForProvider = func(string) local_packages_parser.LocalPackageRoot {
		return local_packages_parser.LocalPackageRoot{Packages: nil}
	}
	// Force symlink creation error in Install's post-sync step
	oldSym := npmSymlink
	npmSymlink = func(string, string) error { return errors.New("sym") }
	// Call Install with a specific package
	assert.True(t, p.Install("pkg:npm/post", "1.0.0"))
	// restore
	npmSymlink = oldSym
	lppGetDataForProvider = oldGetProv
}

func TestNPMRemoveLogsSymlinkRemovalError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// Prepare node_modules/pkg with a bin so removePackageSymlinks will try to remove
	binDir := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin")
	_ = os.MkdirAll(binDir, 0755)
	pkgDir := filepath.Join(p.APP_PACKAGES_DIR, "node_modules", "pkg")
	_ = os.MkdirAll(pkgDir, 0755)
	_ = os.WriteFile(filepath.Join(pkgDir, "package.json"), []byte(`{"name":"pkg","version":"1.0.0","bin":{"cli":"./bin/cli.js"}}`), 0644)
	// Force symlink removal error and ensure removePackageSymlinks returns error to trigger log in Remove
	oldLs, oldRm := npmLstat, npmRemove
	npmLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	npmRemove = func(string) error { return errors.New("rm") }
	// Ensure local removal succeeds
	oldLR := lppRemove
	lppRemove = func(string) error { return nil }
	assert.True(t, p.Remove("pkg:npm/pkg"))
	lppRemove = oldLR
	npmLstat, npmRemove = oldLs, oldRm
}

func TestNPMInstallIndividualPathSymlinkError(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderNPM()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired d@1.0.0 not installed
	_ = lppAdd("pkg:npm/d", "1.0.0")
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "node_modules", ".bin"), 0755)
	// install succeeds
	oldOut := npmShellOut
	npmShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	assert.True(t, p.Sync())
	npmShellOut = oldOut
}

func TestPyPiSyncMixedInstalledSkippedAndGuiScriptsAndRemoveHappy(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderPyPi()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	_ = os.MkdirAll(files.GetAppBinPath(), 0755)

	// desired c1==1.0.0 (installed) and c2==2.0.0 (to install)
	_ = lppPyAdd("pkg:pypi/c1", "1.0.0")
	_ = lppPyAdd("pkg:pypi/c2", "2.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "c1", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/c1"},
		Bin: map[string]string{"c1": "c1"},
	}, {
		Name: "c2", Version: "2.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/c2"},
		Bin: map[string]string{"c2": "c2"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// freeze shows both installed to take the all-installed fast path and avoid real installs
	oldCap := pipShellOutCapture
	oldOut := pipShellOut
	oldCh := pipChmod
	pipShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "c1==1.0.0\nc2==2.0.0", nil }
	// ensure any pip installs (if reached) succeed; and chmod succeeds
	pipShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	pipChmod = func(string, os.FileMode) error { return nil }
	assert.True(t, p.Sync())

	// parseEntryPoints gui_scripts
	infoDir := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages", "gui-1.0.0.dist-info")
	_ = os.MkdirAll(infoDir, 0755)
	_ = os.WriteFile(filepath.Join(infoDir, "entry_points.txt"), []byte("[gui_scripts]\nrun = r:m\n"), 0644)
	entries := p.parseEntryPointsFromInfoDir(infoDir)
	assert.Contains(t, entries, "run")

	// Remove happy-path with wrappers present
	_ = lppPyAdd("pkg:pypi/tool", "1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "tool", Version: "1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:pypi/tool"},
		Bin: map[string]string{"tool": "tool"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// create wrapper and bin to be removed
	_ = os.WriteFile(filepath.Join(files.GetAppBinPath(), "tool"), []byte(""), 0755)
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages"), 0755)
	pid := filepath.Join(p.APP_PACKAGES_DIR, "lib", "python3.11", "site-packages", "tool-1.0.0.dist-info")
	_ = os.MkdirAll(pid, 0755)
	_ = os.WriteFile(filepath.Join(pid, "entry_points.txt"), []byte("[console_scripts]\ntool = t:m\n"), 0644)
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "bin", "tool"), []byte(""), 0755)
	assert.True(t, p.Remove("pkg:pypi/tool"))
	// restore after Remove's internal Sync completed
	pipChmod = oldCh
	pipShellOut = oldOut
	pipShellOutCapture = oldCap
}

func TestGolangCleanMultipleAndInstallLatestSuccess(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderGolang()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// add two packages
	_ = lppGoAdd("pkg:golang/github.com/a/one", "v1.0.0")
	_ = lppGoAdd("pkg:golang/github.com/a/two", "v1.0.0")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "one", Version: "v1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/a/one"},
		Bin: map[string]string{"one": "one"},
	}, {
		Name: "two", Version: "v1.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/a/two"},
		Bin: map[string]string{"two": "two"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	gobin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(gobin, 0755)
	_ = os.WriteFile(filepath.Join(gobin, "one"), []byte(""), 0755)
	_ = os.WriteFile(filepath.Join(gobin, "two"), []byte(""), 0755)
	assert.True(t, p.Clean())

	// Install latest success
	_ = lppGoAdd("pkg:golang/github.com/a/inst", "latest")
	writeRegistry(t, []registry_parser.RegistryItem{{
		Name: "inst", Version: "v2.0.0", Source: registry_parser.RegistryItemSource{ID: "pkg:golang/github.com/a/inst"},
		Bin: map[string]string{"inst": "inst"},
	}})
	_ = registry_parser.NewDefaultRegistryParser().GetData(true)
	// pre-create binary so createSymlink can succeed
	_ = os.WriteFile(filepath.Join(gobin, "inst"), []byte(""), 0755)
	oldCap := goShellOutCapture
	oldOut := goShellOut
	goShellOutCapture = func(string, []string, string, []string) (int, string, error) { return 0, "mod v1.0.0 v2.0.0", nil }
	goShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	assert.True(t, p.Install("pkg:golang/github.com/a/inst", "latest"))
	goShellOut = oldOut
	goShellOutCapture = oldCap
}

func TestCargoSyncMixedDesiredInstalledAndRemoveAllSymlinksSuccessAndWarning(t *testing.T) {
	_ = withTempZanaHome(t)
	p := NewProviderCargo()
	_ = os.MkdirAll(p.APP_PACKAGES_DIR, 0755)
	// desired alpha fixed, beta latest
	_ = lppCargoAdd("pkg:cargo/alpha", "1.0.0")
	_ = lppCargoAdd("pkg:cargo/beta", "latest")
	// getInstalledCrates shows alpha installed
	oldCap := cargoShellOutCapture
	cargoShellOutCapture = func(cmd string, args []string, dir string, env []string) (int, string, error) {
		if len(args) > 0 && args[0] == "install" && len(args) > 1 && args[1] == "--list" { // not used
			return 0, "", nil
		}
		if len(args) > 0 && args[0] == "install" {
			return 0, "", nil
		}
		if len(args) > 0 && args[0] == "search" {
			return 0, "beta = \"2.3.4\"", nil
		}
		// default for getInstalledCrates
		return 0, "alpha v1.0.0:", nil
	}
	oldOut := cargoShellOut
	cargoShellOut = func(string, []string, string, []string) (int, error) { return 0, nil }
	// ensure cargo bin exists with some binary for symlink creation call after Sync
	_ = os.MkdirAll(filepath.Join(p.APP_PACKAGES_DIR, "bin"), 0755)
	_ = os.WriteFile(filepath.Join(p.APP_PACKAGES_DIR, "bin", "dummy"), []byte(""), 0755)
	assert.True(t, p.Sync())
	cargoShellOut = oldOut
	cargoShellOutCapture = oldCap

	// removeAllSymlinks success and warning paths
	zbin := files.GetAppBinPath()
	cbin := filepath.Join(p.APP_PACKAGES_DIR, "bin")
	_ = os.MkdirAll(zbin, 0755)
	_ = os.MkdirAll(cbin, 0755)
	// one symlink removable successfully
	sl1 := filepath.Join(zbin, "s1")
	_ = os.Symlink(filepath.Join(cbin, "s1"), sl1)
	// one symlink remove warning
	sl2 := filepath.Join(zbin, "s2")
	_ = os.Symlink(filepath.Join(cbin, "s2"), sl2)
	oldLs := cargoLstat
	oldRl := cargoReadlink
	oldRm := cargoRemove
	cargoLstat = func(string) (os.FileInfo, error) { return fileInfoNow(t), nil }
	cargoReadlink = func(path string) (string, error) {
		if path == sl1 {
			return filepath.Join(cbin, "s1"), nil
		}
		return filepath.Join(cbin, "s2"), nil
	}
	cargoRemove = func(path string) error {
		if path == sl2 {
			return errors.New("rm")
		}
		return nil
	}
	assert.NoError(t, p.removeAllSymlinks())
	cargoLstat, cargoReadlink, cargoRemove = oldLs, oldRl, oldRm
}
