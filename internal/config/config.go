package config

import (
	"fmt"
	"time"
)

type ColorMode string

const (
	ColorModeAuto   ColorMode = "auto"   // Use colors only when TTY
	ColorModeAlways ColorMode = "always" // Always use colors/icons
	ColorModeNever  ColorMode = "never"  // Never use colors/icons
)

// String implements the flag.Value interface for ColorMode
func (c *ColorMode) String() string {
	if c == nil || *c == "" {
		return string(ColorModeAuto)
	}
	return string(*c)
}

// Set implements the flag.Value interface for ColorMode
func (c *ColorMode) Set(value string) error {
	switch value {
	case "always", "auto", "never":
		*c = ColorMode(value)
		return nil
	default:
		return fmt.Errorf("invalid color mode: %s (must be 'always', 'auto', or 'never')", value)
	}
}

// Type implements the flag.Value interface for ColorMode
func (c *ColorMode) Type() string {
	return "string"
}

type ConfigFlags struct {
	Version     bool
	CacheMaxAge time.Duration
	Color       ColorMode
}

type Config struct {
	Flags ConfigFlags
}

func (c Config) GetConfigFlags() ConfigFlags {
	return c.Flags
}

func NewConfig(cfg Config) Config {
	return cfg
}
