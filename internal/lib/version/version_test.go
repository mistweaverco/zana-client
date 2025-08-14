package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	t.Run("version variable exists", func(t *testing.T) {
		// The VERSION variable should exist and be accessible
		assert.IsType(t, "", VERSION)
	})
}
