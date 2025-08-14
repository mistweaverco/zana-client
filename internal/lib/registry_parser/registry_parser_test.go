package registry_parser

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockFileReader implements FileReader for testing
type mockFileReader struct {
	data map[string][]byte
	err  error
}

func (m *mockFileReader) ReadFile(filename string) ([]byte, error) {
	if m.err != nil {
		return nil, m.err
	}
	if data, exists := m.data[filename]; exists {
		return data, nil
	}
	return nil, errors.New("file not found")
}

func TestRegistryItemSource(t *testing.T) {
	t.Run("registry item source structure", func(t *testing.T) {
		source := RegistryItemSource{ID: "pkg:npm/test-package"}
		assert.Equal(t, "pkg:npm/test-package", source.ID)
	})
}

func TestRegistryItem(t *testing.T) {
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
}

func TestNewRegistryParser(t *testing.T) {
	t.Run("creates new registry parser with file reader", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		assert.NotNil(t, parser)
		assert.Equal(t, mockReader, parser.fileReader)
		assert.Empty(t, parser.data)
		assert.False(t, parser.hasData)
	})
}

func TestNewDefaultRegistryParser(t *testing.T) {
	t.Run("creates registry parser with default dependencies", func(t *testing.T) {
		parser := NewDefaultRegistryParser()

		assert.NotNil(t, parser)
		assert.NotNil(t, parser.fileReader)
		assert.Empty(t, parser.data)
		assert.False(t, parser.hasData)
	})
}

func TestGetData(t *testing.T) {
	t.Run("returns empty data when no data loaded", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		data := parser.GetData(false)
		assert.Empty(t, data)
		assert.True(t, parser.hasData)
	})

	t.Run("returns cached data when force is false and data exists", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// First call loads data
		parser.GetData(false)

		// Second call should return cached data
		data := parser.GetData(false)
		assert.Empty(t, data)
		assert.True(t, parser.hasData)
	})

	t.Run("forces refresh when force is true", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// First call loads data
		parser.GetData(false)

		// Force refresh
		data := parser.GetData(true)
		assert.Empty(t, data)
		assert.True(t, parser.hasData)
	})
}

func TestLoadFromBytes(t *testing.T) {
	t.Run("loads and sorts data from valid JSON", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		jsonData := `[
			{"name": "z", "version": "1.0.0", "source": {"id": "pkg:npm/z"}},
			{"name": "a", "version": "2.0.0", "source": {"id": "pkg:npm/a"}}
		]`

		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		data := parser.GetDataForTesting()
		assert.Len(t, data, 2)
		// Should be sorted by name
		assert.Equal(t, "a", data[0].Name)
		assert.Equal(t, "z", data[1].Name)
		assert.True(t, parser.HasDataForTesting())
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		invalidJSON := `{"invalid": json}`

		err := parser.LoadFromBytes([]byte(invalidJSON))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse registry data")
		assert.False(t, parser.HasDataForTesting())
	})

	t.Run("handles empty JSON array", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		emptyJSON := `[]`

		err := parser.LoadFromBytes([]byte(emptyJSON))
		require.NoError(t, err)

		data := parser.GetDataForTesting()
		assert.Empty(t, data)
		assert.True(t, parser.HasDataForTesting())
	})
}

func TestLoadFromFile(t *testing.T) {
	t.Run("loads data from file successfully", func(t *testing.T) {
		jsonData := `[
			{"name": "test", "version": "1.0.0", "source": {"id": "pkg:npm/test"}}
		]`

		mockReader := &mockFileReader{
			data: map[string][]byte{
				"registry.json": []byte(jsonData),
			},
		}

		parser := NewRegistryParser(mockReader)

		err := parser.LoadFromFile("registry.json")
		require.NoError(t, err)

		data := parser.GetDataForTesting()
		assert.Len(t, data, 1)
		assert.Equal(t, "test", data[0].Name)
		assert.True(t, parser.HasDataForTesting())
	})

	t.Run("returns error when file read fails", func(t *testing.T) {
		mockReader := &mockFileReader{
			err: errors.New("file read error"),
		}

		parser := NewRegistryParser(mockReader)

		err := parser.LoadFromFile("registry.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read registry file")
		assert.False(t, parser.HasDataForTesting())
	})

	t.Run("returns error when file not found", func(t *testing.T) {
		mockReader := &mockFileReader{
			data: map[string][]byte{},
		}

		parser := NewRegistryParser(mockReader)

		err := parser.LoadFromFile("nonexistent.json")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")
		assert.False(t, parser.HasDataForTesting())
	})
}

func TestGetBySourceId(t *testing.T) {
	t.Run("finds item by source ID", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// Load some test data
		jsonData := `[
			{"name": "test1", "version": "1.0.0", "source": {"id": "pkg:npm/test1"}},
			{"name": "test2", "version": "2.0.0", "source": {"id": "pkg:npm/test2"}}
		]`

		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		item := parser.GetBySourceId("pkg:npm/test2")
		assert.Equal(t, "test2", item.Name)
		assert.Equal(t, "2.0.0", item.Version)
	})

	t.Run("returns empty item when source ID not found", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// Load some test data
		jsonData := `[
			{"name": "test1", "version": "1.0.0", "source": {"id": "pkg:npm/test1"}}
		]`

		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		item := parser.GetBySourceId("pkg:npm/nonexistent")
		assert.Equal(t, RegistryItem{}, item)
	})

	t.Run("returns empty item when no data loaded", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		item := parser.GetBySourceId("pkg:npm/test")
		assert.Equal(t, RegistryItem{}, item)
	})
}

func TestGetLatestVersion(t *testing.T) {
	t.Run("returns version when item found", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// Load some test data
		jsonData := `[
			{"name": "test", "version": "1.5.0", "source": {"id": "pkg:npm/test"}}
		]`

		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		version := parser.GetLatestVersion("pkg:npm/test")
		assert.Equal(t, "1.5.0", version)
	})

	t.Run("returns empty string when item not found", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// Load some test data
		jsonData := `[
			{"name": "test", "version": "1.0.0", "source": {"id": "pkg:npm/test"}}
		]`

		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		version := parser.GetLatestVersion("pkg:npm/nonexistent")
		assert.Equal(t, "", version)
	})

	t.Run("returns empty string when no data loaded", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		version := parser.GetLatestVersion("pkg:npm/test")
		assert.Equal(t, "", version)
	})
}

func TestGetDataForTesting(t *testing.T) {
	t.Run("returns current data", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// Initially empty
		data := parser.GetDataForTesting()
		assert.Empty(t, data)

		// Load some data
		jsonData := `[{"name": "test", "version": "1.0.0", "source": {"id": "pkg:npm/test"}}]`
		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		// Now should have data
		data = parser.GetDataForTesting()
		assert.Len(t, data, 1)
		assert.Equal(t, "test", data[0].Name)
	})
}

func TestHasDataForTesting(t *testing.T) {
	t.Run("returns correct data state", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// Initially false
		assert.False(t, parser.HasDataForTesting())

		// Load some data
		jsonData := `[{"name": "test", "version": "1.0.0", "source": {"id": "pkg:npm/test"}}]`
		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		// Now should be true
		assert.True(t, parser.HasDataForTesting())
	})
}

func TestDefaultFileReader(t *testing.T) {
	t.Run("default file reader returns error for non-existent file", func(t *testing.T) {
		reader := &defaultFileReader{}

		data, err := reader.ReadFile("nonexistent.json")
		assert.Nil(t, data)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestIntegration(t *testing.T) {
	t.Run("full workflow with sorting", func(t *testing.T) {
		mockReader := &mockFileReader{}
		parser := NewRegistryParser(mockReader)

		// Load unsorted data
		jsonData := `[
			{"name": "zebra", "version": "3.0.0", "source": {"id": "pkg:npm/zebra"}},
			{"name": "alpha", "version": "1.0.0", "source": {"id": "pkg:npm/alpha"}},
			{"name": "beta", "version": "2.0.0", "source": {"id": "pkg:npm/beta"}}
		]`

		err := parser.LoadFromBytes([]byte(jsonData))
		require.NoError(t, err)

		// Verify data is sorted
		data := parser.GetDataForTesting()
		assert.Len(t, data, 3)
		assert.Equal(t, "alpha", data[0].Name)
		assert.Equal(t, "beta", data[1].Name)
		assert.Equal(t, "zebra", data[2].Name)

		// Test finding items
		alpha := parser.GetBySourceId("pkg:npm/alpha")
		assert.Equal(t, "alpha", alpha.Name)
		assert.Equal(t, "1.0.0", alpha.Version)

		beta := parser.GetBySourceId("pkg:npm/beta")
		assert.Equal(t, "beta", beta.Name)
		assert.Equal(t, "2.0.0", beta.Version)

		zebra := parser.GetBySourceId("pkg:npm/zebra")
		assert.Equal(t, "zebra", zebra.Name)
		assert.Equal(t, "3.0.0", zebra.Version)

		// Test version retrieval
		assert.Equal(t, "1.0.0", parser.GetLatestVersion("pkg:npm/alpha"))
		assert.Equal(t, "2.0.0", parser.GetLatestVersion("pkg:npm/beta"))
		assert.Equal(t, "3.0.0", parser.GetLatestVersion("pkg:npm/zebra"))
	})
}
