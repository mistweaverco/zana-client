package zana

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/files"
)

// TestMain isolates command tests from the user's real filesystem.
// Many tests touch registry/local package paths via files.GetApp* helpers.
func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "zana-test-home-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	// Force all zana paths under this temp dir.
	_ = os.Setenv("ZANA_HOME", tmp)

	// Ensure expected dirs exist (avoids warnings in some code paths).
	_ = files.GetAppDataPath()
	_ = files.GetAppBinPath()
	_ = files.GetAppPackagesPath()
	_ = os.MkdirAll(filepath.Join(tmp, "cache"), 0755)

	os.Exit(m.Run())
}
