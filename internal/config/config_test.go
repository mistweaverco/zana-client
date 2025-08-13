package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	t.Run("new config creation", func(t *testing.T) {
		flags := ConfigFlags{
			Version:     true,
			CacheMaxAge: 24 * time.Hour,
		}
		cfg := NewConfig(Config{Flags: flags})

		assert.Equal(t, true, cfg.Flags.Version)
		assert.Equal(t, 24*time.Hour, cfg.Flags.CacheMaxAge)
	})

	t.Run("get config flags", func(t *testing.T) {
		flags := ConfigFlags{
			Version:     false,
			CacheMaxAge: 12 * time.Hour,
		}
		cfg := Config{Flags: flags}

		result := cfg.GetConfigFlags()
		assert.Equal(t, false, result.Version)
		assert.Equal(t, 12*time.Hour, result.CacheMaxAge)
	})

	t.Run("config flags default values", func(t *testing.T) {
		var flags ConfigFlags
		cfg := Config{Flags: flags}

		result := cfg.GetConfigFlags()
		assert.Equal(t, false, result.Version)
		assert.Equal(t, time.Duration(0), result.CacheMaxAge)
	})
}
