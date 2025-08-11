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
)

var PS = string(os.PathSeparator)

func Download(url string, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Log the error but don't fail the function
			fmt.Printf("Warning: failed to close response body: %v\n", closeErr)
		}
	}()

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
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
	return GetAppDataPath() + PS + "zana-lock.json"
}

func FileExists(path string) bool {
	_, err :=
		os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

// GenerateZanaGitIgnore creates a .gitignore file at the top level of the zana config directory
// if it doesn't exist. The .gitignore ignores *.zip files and the /bin directory.
func GenerateZanaGitIgnore() bool {
	configDir := GetAppDataPath()
	gitignorePath := configDir + PS + ".gitignore"

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

	file, err := os.Create(gitignorePath)
	if err != nil {
		fmt.Println("Error creating .gitignore:", err)
		return false
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			fmt.Printf("Warning: failed to close .gitignore file: %v\n", closeErr)
		}
	}()

	_, err = file.WriteString(contents)
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
	if zanaHome := os.Getenv("ZANA_HOME"); zanaHome != "" {
		return EnsureDirExists(zanaHome)
	}
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}
	return EnsureDirExists(userConfigDir + PS + "zana")
}

// GetTempPath returns the path to the temp directory
// e.g. /tmp
func GetTempPath() string {
	return os.TempDir()
}

// GetAppRegistryFilePath returns the path to the registry file
// e.g. /home/user/.config/zana/zana-registry.json
func GetAppRegistryFilePath() string {
	return GetAppDataPath() + PS + "zana-registry.json"
}

// GetAppPackagesPath returns the path to the packages directory
// e.g. /home/user/.config/zana/packages
func GetAppPackagesPath() string {
	return EnsureDirExists(GetAppDataPath() + PS + "packages")
}

// GetAppBinPath returns the path to the bin directory
// e.g. /home/user/.config/zana/bin
func GetAppBinPath() string {
	path := GetAppDataPath() + PS + "bin"
	return EnsureDirExists(path)
}

func EnsureDirExists(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			// Log the error but don't fail the function
			fmt.Printf("Warning: failed to create directory %s: %v\n", path, err)
		}
	}
	return path
}

func Unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	if err := os.MkdirAll(dest, 0755); err != nil {
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
		if !strings.HasPrefix(path, filepath.Clean(dest)+PS) {
			return fmt.Errorf("illegal file path: %s", path)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", path, err)
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(path), f.Mode()); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(path), err)
			}
			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
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

	for _, f := range r.File {
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
	return GetAppDataPath() + PS + "registry-cache.json.zip"
}

// IsCacheValid checks if the cache file exists and is newer than the specified duration
func IsCacheValid(cachePath string, maxAge time.Duration) bool {
	fileInfo, err := os.Stat(cachePath)
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
	resp, err := http.Get(url)
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
	out, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil {
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
	if override := os.Getenv("ZANA_REGISTRY_URL"); override != "" {
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
