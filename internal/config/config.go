package config

import "time"

type ConfigFlags struct {
	Version     bool
	CacheMaxAge time.Duration
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
