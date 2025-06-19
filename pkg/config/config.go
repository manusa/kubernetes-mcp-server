package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type StaticConfig struct {
	DeniedResources []GroupVersionKind `toml:"denied_resources"`
}

type GroupVersionKind struct {
	Group   string `toml:"group"`
	Version string `toml:"version"`
	Kind    string `toml:"kind,omitempty"`
}

func ReadConfig(configPath string) (*StaticConfig, error) {
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var config *StaticConfig
	err = toml.Unmarshal(configData, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
