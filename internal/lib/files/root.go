package files

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// FileSystem interface for filesystem operations
type FileSystem interface {
	Create(name string) (afero.File, error)
	MkdirAll(path string, perm os.FileMode) error
	OpenFile(name string, flag int, perm os.FileMode) (afero.File, error)
	Stat(name string) (os.FileInfo, error)
	UserConfigDir() (string, error)
	TempDir() string
	Getenv(key string) string
	WriteString(file afero.File, s string) (int, error)
	Close(file afero.File) error
}

// HTTPClient interface for HTTP operations
type HTTPClient interface {
	Get(url string) (*http.Response, error)
}

// ZipReader interface for zip operations
type ZipReader interface {
	OpenReader(name string) (*zip.ReadCloser, error)
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

// defaultZipReader implements ZipReader
type defaultZipReader struct{}

func (d *defaultZipReader) OpenReader(name string) (*zip.ReadCloser, error) {
	return zip.OpenReader(name)
}

// zipReaderAdapter adapts the old ZipReader interface to the new ZipFileOpener interface
type zipReaderAdapter struct {
	zipReader ZipReader
}

func (a *zipReaderAdapter) Open(name string) (ZipArchive, error) {
	rc, err := a.zipReader.OpenReader(name)
	if err != nil {
		return nil, err
	}
	return &RealZipArchive{ReadCloser: rc}, nil
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
	zipReader     ZipReader     = &defaultZipReader{}
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

// SetZipReader sets the zip reader implementation
func SetZipReader(zr ZipReader) {
	zipReader = zr
	// Also set the zipFileOpener using the adapter for backward compatibility
	zipFileOpener = &zipReaderAdapter{zipReader: zr}
}

// SetZipFileOpener sets the zip file opener implementation
func SetZipFileOpener(zfo ZipFileOpener) {
	zipFileOpener = zfo
}

// ResetDependencies resets all dependencies to their default implementations
func ResetDependencies() {
	fileSystem = &defaultFileSystem{fs: afero.NewOsFs()}
	httpClient = &defaultHTTPClient{}
	zipReader = &defaultZipReader{}
	zipFileOpener = &zipReaderAdapter{zipReader: &defaultZipReader{}}
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

// GenerateZanaGitIgnore creates a .gitignore file at the top level of the zana config directory
// if it doesn't exist. The .gitignore ignores *.zip files and the /bin directory.
func GenerateZanaGitIgnore() bool {
	configDir := GetAppDataPath()
	gitignorePath := configDir + string(os.PathSeparator) + ".gitignore"

	if FileExists(gitignorePath) {
		return true
	}

	contents := `# Zana configuration directory .gitignore
# Ignore zip files
*.zip

# Ignore zana-registry file
zana-registry.json

# Ignore bin directory
/bin

# Ignore packages directory
/packages

# Ignore other common temporary files
*.tmp
*.log
`

	file, err := fileSystem.Create(gitignorePath)
	if err != nil {
		fmt.Println("Error creating .gitignore:", err)
		return false
	}
	defer func() {
		if closeErr := fileSystem.Close(file); closeErr != nil {
			fmt.Printf("Warning: failed to close .gitignore file: %v\n", closeErr)
		}
	}()

	_, err = fileSystem.WriteString(file, contents)
	if err != nil {
		fmt.Println("Error writing to .gitignore:", err)
		return false
	}

	fmt.Println("Created .gitignore in zana config directory")
	return true
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
// e.g. /home/user/.config/zana/zana-registry.json
func GetAppRegistryFilePath() string {
	return GetAppDataPath() + string(os.PathSeparator) + "zana-registry.json"
}

// GetAppPackagesPath returns the path to the packages directory
// e.g. /home/user/.config/zana/packages
func GetAppPackagesPath() string {
	return EnsureDirExists(GetAppDataPath() + string(os.PathSeparator) + "packages")
}

// GetAppBinPath returns the path to the bin directory
// e.g. /home/user/.config/zana/bin
func GetAppBinPath() string {
	path := GetAppDataPath() + string(os.PathSeparator) + "bin"
	return EnsureDirExists(path)
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

// GetRegistryCachePath returns the path to the registry cache file
// e.g. /home/user/.config/zana/registry-cache.json.zip
func GetRegistryCachePath() string {
	return GetAppDataPath() + string(os.PathSeparator) + "registry-cache.json.zip"
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

	// Download the registry
	if err := DownloadWithCache(registryURL, cachePath, 24*time.Hour); err != nil {
		return fmt.Errorf("failed to download registry: %w", err)
	}

	// Unzip the registry
	if err := Unzip(cachePath, GetAppDataPath()); err != nil {
		return fmt.Errorf("failed to unzip registry: %w", err)
	}

	return nil
}
