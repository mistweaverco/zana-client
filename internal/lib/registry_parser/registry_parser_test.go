package registry_parser

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/mistweaverco/zana-client/internal/lib/files"
	"github.com/stretchr/testify/assert"
)

func TestRegistryParser(t *testing.T) {
	t.Run("registry item source structure", func(t *testing.T) {
		source := RegistryItemSource{ID: "pkg:npm/test-package"}
		assert.Equal(t, "pkg:npm/test-package", source.ID)
	})

	t.Run("registry item structure", func(t *testing.T) {
		item := RegistryItem{
			Name:        "test-package",
			Version:     "1.0.0",
			Description: "A test package",
			Homepage:    "https://example.com",
			Licenses:    []string{"MIT"},
			Languages:   []string{"JavaScript"},
			Categories:  []string{"testing"},
			Source:      RegistryItemSource{ID: "pkg:npm/test-package"},
			Bin:         map[string]string{"test": "bin/test"},
		}

		assert.Equal(t, "test-package", item.Name)
		assert.Equal(t, "1.0.0", item.Version)
		assert.Equal(t, "A test package", item.Description)
		assert.Equal(t, "https://example.com", item.Homepage)
		assert.Equal(t, []string{"MIT"}, item.Licenses)
		assert.Equal(t, []string{"JavaScript"}, item.Languages)
		assert.Equal(t, []string{"testing"}, item.Categories)
		assert.Equal(t, "pkg:npm/test-package", item.Source.ID)
		assert.Equal(t, "bin/test", item.Bin["test"])
	})

	t.Run("get data without forcing", func(t *testing.T) {
		// This will likely return empty data since no registry file exists in test environment
		data := GetData(false)
		assert.NotNil(t, data)
	})

	t.Run("get by source id with empty registry", func(t *testing.T) {
		// With empty registry, should return empty item
		item := GetBySourceId("pkg:npm/test-package")
		assert.Equal(t, RegistryItem{}, item)
	})

	t.Run("get latest version with empty registry", func(t *testing.T) {
		// With empty registry, should return empty string
		version := GetLatestVersion("pkg:npm/test-package")
		assert.Equal(t, "", version)
	})
}

func TestRegistryParserCoversBranches(t *testing.T) {
	t.Run("GetData returns empty when file missing and caches result", func(t *testing.T) {
		// Ensure no registry file exists in temp home
		t.Setenv("ZANA_HOME", t.TempDir())
		_ = files.GetAppDataPath()
		data := GetData(true)
		assert.Empty(t, data)
		// second call without force uses cache path (still empty)
		data = GetData(false)
		assert.Empty(t, data)
	})

	t.Run("GetData reads, sorts and GetBySourceId works", func(t *testing.T) {
		t.Setenv("ZANA_HOME", t.TempDir())
		_ = files.GetAppDataPath()
		items := RegistryRoot{
			{Name: "z", Version: "1.0.0", Source: RegistryItemSource{ID: "pkg:npm/z"}},
			{Name: "a", Version: "2.0.0", Source: RegistryItemSource{ID: "pkg:npm/a"}},
		}
		b, _ := json.Marshal(items)
		err := os.WriteFile(files.GetAppRegistryFilePath(), b, 0644)
		assert.NoError(t, err)

		out := GetData(true)
		assert.Len(t, out, 2)
		// sorted by name ascending
		assert.Equal(t, "a", out[0].Name)
		assert.Equal(t, "z", out[1].Name)

		item := GetBySourceId("pkg:npm/a")
		assert.Equal(t, "a", item.Name)
		assert.Equal(t, "2.0.0", GetLatestVersion("pkg:npm/a"))
	})
}
