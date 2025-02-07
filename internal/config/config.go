package config

type ConfigFlags struct {
	Version bool
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
