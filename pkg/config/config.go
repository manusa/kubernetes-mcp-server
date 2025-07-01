package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

// StaticConfig is the configuration for the server.
// It allows to configure server specific settings and tools to be enabled or disabled.
type StaticConfig struct {
	DeniedResources []GroupVersionKind `toml:"denied_resources"`

	LogLevel           int      `toml:"log_level,omitempty"`
	SSEPort            int      `toml:"sse_port,omitempty"`
	HTTPPort           int      `toml:"http_port,omitempty"`
	SSEBaseURL         string   `toml:"sse_base_url,omitempty"`
	KubeConfig         string   `toml:"kubeconfig,omitempty"`
	ListOutput         string   `toml:"list_output,omitempty"`
	ReadOnly           bool     `toml:"read_only,omitempty"`
	DisableDestructive bool     `toml:"disable_destructive,omitempty"`
	AllowedTools       []string `toml:"allowed_tools,omitempty"`
	DeniedTools        []string `toml:"denied_tools,omitempty"`
}

type GroupVersionKind struct {
	Group   string `toml:"group"`
	Version string `toml:"version"`
	Kind    string `toml:"kind,omitempty"`
}

// ReadConfig reads the toml file and returns the StaticConfig.
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
