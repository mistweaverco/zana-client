package files

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

// GetAppDataPath returns the path to the app data directory
// e.g. /home/user/.config/zana
func GetAppDataPath() string {
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
