package files

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/huh/spinner"
	"github.com/spf13/afero"
)

// FileSystem interface for filesystem operations
type FileSystem interface {
	Create(name string) (afero.File, error)
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (afero.File, error)
	Stat(name string) (os.FileInfo, error)
	UserConfigDir() (string, error)
	UserHomeDir() (string, error)
	TempDir() string
	Getenv(key string) string
	WriteString(file afero.File, s string) (int, error)
	Close(file afero.File) error
}

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// ZipArchive is an interface that abstracts the functionality
// of a *zip.ReadCloser.
type ZipArchive interface {
	File() []*zip.File
	Close() error
}

// ZipFileOpener is the interface for opening a zip file.
type ZipFileOpener interface {
	Open(name string) (ZipArchive, error)
}

// defaultFileSystem implements FileSystem using Afero
type defaultFileSystem struct {
	fs afero.Fs
}

func (d *defaultFileSystem) Create(name string) (afero.File, error) {
	return d.fs.Create(name)
}

func (d *defaultFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return d.fs.MkdirAll(path, perm)
}

func (d *defaultFileSystem) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return d.fs.OpenFile(name, flag, perm)
}

func (d *defaultFileSystem) Stat(name string) (os.FileInfo, error) {
	return d.fs.Stat(name)
}

func (d *defaultFileSystem) UserConfigDir() (string, error) {
	return os.UserConfigDir()
}

func (d *defaultFileSystem) UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func (d *defaultFileSystem) TempDir() string {
	return os.TempDir()
}

func (d *defaultFileSystem) Getenv(key string) string {
	return os.Getenv(key)
}

func (d *defaultFileSystem) WriteString(file afero.File, s string) (int, error) {
	return file.WriteString(s)
}

func (d *defaultFileSystem) Close(file afero.File) error {
	return file.Close()
}

// defaultHTTPClient implements HTTPClient
type defaultHTTPClient struct{}

func (d *defaultHTTPClient) Get(url string) (*http.Response, error) {
	return http.Get(url)
}

// RealZipArchive is a wrapper for a real *zip.ReadCloser
type RealZipArchive struct {
	*zip.ReadCloser
}

// File provides access to the embedded zip.ReadCloser's File field
func (r *RealZipArchive) File() []*zip.File {
	return r.ReadCloser.File
}

// Close closes the underlying zip.ReadCloser
func (r *RealZipArchive) Close() error {
	return r.ReadCloser.Close()
}

// RealZipFileOpener implements the ZipFileOpener interface using the real zip package.
type RealZipFileOpener struct{}

func (o *RealZipFileOpener) Open(name string) (ZipArchive, error) {
	rc, err := zip.OpenReader(name)
	if err != nil {
		return nil, err
	}
	return &RealZipArchive{rc}, nil
}

// Global variables for dependency injection
var (
	fileSystem    FileSystem    = &defaultFileSystem{fs: afero.NewOsFs()}
	httpClient    HTTPClient    = &defaultHTTPClient{}
	zipFileOpener ZipFileOpener = &RealZipFileOpener{}
)

// SetFileSystem sets the file system implementation
func SetFileSystem(fs FileSystem) {
	fileSystem = fs
}

// SetHTTPClient sets the HTTP client implementation
func SetHTTPClient(client HTTPClient) {
	httpClient = client
}

// SetZipFileOpener sets the zip file opener implementation
func SetZipFileOpener(zfo ZipFileOpener) {
	zipFileOpener = zfo
}

// ResetDependencies resets all dependencies to their default implementations
func ResetDependencies() {
	fileSystem = &defaultFileSystem{fs: afero.NewOsFs()}
	httpClient = &defaultHTTPClient{}
	zipFileOpener = &RealZipFileOpener{}
}

func Download(url string, dest string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't fail the function
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	out, err := fileSystem.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fileSystem.Close(out); closeErr != nil {
			// Log the error but don't fail the function
			fmt.Printf("Warning: failed to close output file: %v\n", closeErr)
		}
	}()

	_, err = io.Copy(out, resp.Body)
	return err
}

// GetAppLocalPackagesFilePath returns the path to the local packages file
// e.g. /home/user/.config/zana/zana-lock.json
func GetAppLocalPackagesFilePath() string {
	return GetAppDataPath() + string(os.PathSeparator) + "zana-lock.json"
}

func FileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err :=
		fileSystem.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// GetAppDataPath returns the path to the app data directory
// If the ZANA_HOME environment variable is set, it will use that path
// otherwise it will use the user's config directory
// e.g. /home/user/.config/zana
func GetAppDataPath() string {
	if zanaHome := fileSystem.Getenv("ZANA_HOME"); zanaHome != "" {
		return EnsureDirExists(zanaHome)
	}
	userConfigDir, err := fileSystem.UserConfigDir()
	if err != nil {
		panic(err)
	}
	return EnsureDirExists(userConfigDir + string(os.PathSeparator) + "zana")
}

// GetTempPath returns the path to the temp directory
// e.g. /tmp
func GetTempPath() string {
	return fileSystem.TempDir()
}

// GetAppRegistryFilePath returns the path to the registry file
// e.g. /home/user/.cache/zana/zana-registry.json
func GetAppRegistryFilePath() string {
	return GetCachePath() + string(os.PathSeparator) + "zana-registry.json"
}

// GetAppPackagesPath returns the path to the packages directory
// If ZANA_HOME is set, it will use $ZANA_HOME/packages
// Otherwise:
//   - Linux: ~/.local/share/zana/packages
//   - macOS: ~/Library/Application Support/zana/packages
//   - Windows: %APPDATA%\zana\packages
func GetAppPackagesPath() string {
	if zanaHome := fileSystem.Getenv("ZANA_HOME"); zanaHome != "" {
		return EnsureDirExists(zanaHome + string(os.PathSeparator) + "packages")
	}
	return EnsureDirExists(GetAppDataSharePath() + string(os.PathSeparator) + "packages")
}

// GetAppDataSharePath returns the path to the app data share directory
// This is separate from the config directory and follows XDG Base Directory spec
// If ZANA_HOME is set, it will use that path
// Otherwise:
//   - Linux: ~/.local/share/zana
//   - macOS: ~/Library/Application Support/zana (same as config)
//   - Windows: %APPDATA%\zana (same as config)
func GetAppDataSharePath() string {
	if zanaHome := fileSystem.Getenv("ZANA_HOME"); zanaHome != "" {
		return EnsureDirExists(zanaHome)
	}

	// On Linux, use ~/.local/share, otherwise use config dir (macOS/Windows)
	userConfigDir, err := fileSystem.UserConfigDir()
	if err != nil {
		panic(err)
	}

	// Check if we're on Linux by checking if config dir ends with .config
	// On Linux: ~/.config, on macOS: ~/Library/Application Support, on Windows: %APPDATA%
	if strings.Contains(userConfigDir, ".config") {
		// Linux: use ~/.local/share instead of ~/.config
		userHomeDir, err := fileSystem.UserHomeDir()
		if err != nil {
			panic(err)
		}
		return EnsureDirExists(userHomeDir + string(os.PathSeparator) + ".local" + string(os.PathSeparator) + "share" + string(os.PathSeparator) + "zana")
	}

	// macOS and Windows: use config directory (same location)
	return EnsureDirExists(userConfigDir + string(os.PathSeparator) + "zana")
}

// GetAppBinPath returns the path to the bin directory
// If ZANA_HOME is set, it will use $ZANA_HOME/bin
// Otherwise:
//   - Linux: ~/.local/share/zana/bin
//   - macOS: ~/Library/Application Support/zana/bin
//   - Windows: %APPDATA%\zana\bin
//
// e.g. /home/user/.local/share/zana/bin
func GetAppBinPath() string {
	if zanaHome := fileSystem.Getenv("ZANA_HOME"); zanaHome != "" {
		return EnsureDirExists(zanaHome + string(os.PathSeparator) + "bin")
	}
	return EnsureDirExists(GetAppDataSharePath() + string(os.PathSeparator) + "bin")
}

func EnsureDirExists(path string) string {
	if _, err := fileSystem.Stat(path); os.IsNotExist(err) {
		if err := fileSystem.MkdirAll(path, 0755); err != nil {
			// Log the error but don't fail the function
			fmt.Printf("Warning: failed to create directory %s: %v\n", path, err)
		}
	}
	return path
}

func Unzip(src, dest string) error {
	r, err := zipFileOpener.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	if err := fileSystem.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		// Check for ZipSlip (Directory traversal)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			if err := fileSystem.MkdirAll(path, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
		} else {
			if err := fileSystem.MkdirAll(filepath.Dir(path), f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(path), err)
			}
			f, err := fileSystem.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := fileSystem.Close(f); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File() {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetCachePath returns the path to the cache directory
// If ZANA_CACHE is set, it will use that path
// Otherwise:
//   - Linux: ~/.cache/zana
//   - macOS: ~/Library/Caches/zana
//   - Windows: %LOCALAPPDATA%\zana\cache
func GetCachePath() string {
	if zanaCache := fileSystem.Getenv("ZANA_CACHE"); zanaCache != "" {
		return EnsureDirExists(zanaCache)
	}

	userHomeDir, err := fileSystem.UserHomeDir()
	if err != nil {
		panic(err)
	}

	var cacheDir string
	switch runtime.GOOS {
	case "linux":
		// Linux: ~/.cache/zana (XDG Base Directory spec)
		cacheDir = userHomeDir + string(os.PathSeparator) + ".cache" + string(os.PathSeparator) + "zana"
	case "darwin":
		// macOS: ~/Library/Caches/zana
		cacheDir = userHomeDir + string(os.PathSeparator) + "Library" + string(os.PathSeparator) + "Caches" + string(os.PathSeparator) + "zana"
	case "windows":
		// Windows: %LOCALAPPDATA%\zana\cache
		// Try LOCALAPPDATA first, fallback to APPDATA if not set
		localAppData := fileSystem.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			appData := fileSystem.Getenv("APPDATA")
			if appData != "" {
				cacheDir = appData + string(os.PathSeparator) + "zana" + string(os.PathSeparator) + "cache"
			} else {
				// Fallback to user home
				cacheDir = userHomeDir + string(os.PathSeparator) + ".zana" + string(os.PathSeparator) + "cache"
			}
		} else {
			cacheDir = localAppData + string(os.PathSeparator) + "zana" + string(os.PathSeparator) + "cache"
		}
	default:
		// Default: use user home with .cache/zana
		cacheDir = userHomeDir + string(os.PathSeparator) + ".cache" + string(os.PathSeparator) + "zana"
	}

	return EnsureDirExists(cacheDir)
}

// GetRegistryCachePath returns the path to the registry cache file
// e.g. /home/user/.cache/zana/registry-cache.json.zip
func GetRegistryCachePath() string {
	return GetCachePath() + string(os.PathSeparator) + "registry-cache.json.zip"
}

// IsCacheValid checks if the cache file exists and is newer than the specified duration
func IsCacheValid(cachePath string, maxAge time.Duration) bool {
	fileInfo, err := fileSystem.Stat(cachePath)
	if err != nil {
		return false // Cache file doesn't exist
	}

	// Check if the file is older than maxAge
	return time.Since(fileInfo.ModTime()) < maxAge
}

// DownloadWithCache downloads a file with caching support
func DownloadWithCache(url string, cachePath string, maxAge time.Duration) error {
	// Check if cache is valid
	if IsCacheValid(cachePath, maxAge) {
		return nil // Cache is valid, no need to download
	}

	// Download the file
	resp, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't fail the function
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	// Create the cache file
	out, err := fileSystem.Create(cachePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fileSystem.Close(out); closeErr != nil {
			// Log the error but don't fail the function
			fmt.Printf("Warning: failed to close cache file: %v\n", closeErr)
		}
	}()

	// Copy the response to the cache file
	_, err = io.Copy(out, resp.Body)
	return err
}

// DownloadAndUnzipRegistry downloads the registry from the default URL and unzips it
// This is used to ensure the registry is available for commands that need it
func DownloadAndUnzipRegistry() error {
	registryURL := "https://github.com/mistweaverco/zana-registry/releases/latest/download/zana-registry.json.zip"
	if override := fileSystem.Getenv("ZANA_REGISTRY_URL"); override != "" {
		registryURL = override
	}

	cachePath := GetRegistryCachePath()
	registryJSONPath := GetAppRegistryFilePath()

	// Check if cache is valid first
	if IsCacheValid(cachePath, 24*time.Hour) {
		// Cache is valid, check if JSON file exists and unzip if needed
		if !FileExists(registryJSONPath) {
			// Zip exists but JSON doesn't, unzip it
			if err := Unzip(cachePath, GetCachePath()); err != nil {
				return fmt.Errorf("failed to unzip registry: %w", err)
			}
		}
		return nil
	}

	// Download the registry with spinner
	var downloadErr error
	action := func() {
		downloadErr = DownloadWithCache(registryURL, cachePath, 24*time.Hour)
	}

	if err := spinner.New().Title("Downloading registry...").Action(action).Run(); err != nil {
		return err
	}

	if downloadErr != nil {
		return fmt.Errorf("failed to download registry: %w", downloadErr)
	}

	// Unzip the registry to the cache directory
	if err := Unzip(cachePath, GetCachePath()); err != nil {
		return fmt.Errorf("failed to unzip registry: %w", err)
	}

	return nil
}
