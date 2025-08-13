package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSourceParser(t *testing.T) {
	t.Run("parse source with valid format", func(t *testing.T) {
		parser := &SourceParser{}
		provider, packageID := parser.ParseSource("pkg:npm/test-package")

		assert.Equal(t, "npm", provider)
		assert.Equal(t, "test-package", packageID)
	})

	t.Run("parse source with scope", func(t *testing.T) {
		parser := &SourceParser{}
		provider, packageID := parser.ParseSource("pkg:npm/@scope/package-name")

		assert.Equal(t, "npm", provider)
		assert.Equal(t, "@scope", packageID) // Only takes first part after splitting
	})

	t.Run("parse source without pkg prefix", func(t *testing.T) {
		parser := &SourceParser{}
		provider, packageID := parser.ParseSource("npm/test-package")

		assert.Equal(t, "npm", provider)
		assert.Equal(t, "test-package", packageID)
	})

	t.Run("parse source with complex package name", func(t *testing.T) {
		parser := &SourceParser{}
		provider, packageID := parser.ParseSource("pkg:golang/golang.org/x/tools/gopls")

		assert.Equal(t, "golang", provider)
		assert.Equal(t, "golang.org", packageID) // Only takes first part after splitting
	})
}
