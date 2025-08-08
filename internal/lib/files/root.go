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

func Download(url string, dest string) {
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	out, _ := os.Create(dest)
	defer out.Close()
	io.Copy(out, resp.Body)
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

func GenerateGitIgnoreAllFiles(dir string) bool {
	if FileExists(dir + PS + ".gitignore") {
		return true
	}
	EnsureDirExists(dir)
	contents := `# Ignore all files
*
# Except directories
!.gitignore
`
	filePath := dir + PS + ".gitignore"
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating .gitignore:", err)
		return false
	}
	defer file.Close()
	_, err = file.WriteString(contents)
	if err != nil {
		fmt.Println("Error writing to .gitignore:", err)
		return false
	}
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
	GenerateGitIgnoreAllFiles(path)
	return EnsureDirExists(path)
}

func EnsureDirExists(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
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

	os.MkdirAll(dest, 0755)

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
			os.MkdirAll(path, f.Mode())
		} else {
			os.MkdirAll(filepath.Dir(path), f.Mode())
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
	defer resp.Body.Close()

	// Create the cache file
	out, err := os.Create(cachePath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Copy the response to the cache file
	_, err = io.Copy(out, resp.Body)
	return err
}
