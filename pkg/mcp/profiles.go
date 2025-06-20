package mcp

import (
	"embed"
	"io/fs"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/mark3labs/mcp-go/server"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
)

//go:embed configs/full.toml
var defaultFullConfigFile embed.FS

//go:embed configs/full-safe.toml
var defaultFullSafeConfigFile embed.FS

type Profile interface {
	GetName() string
	GetDescription() string
	GetTools(s *Server) []server.ServerTool
	GetDefaultConfig() (*config.StaticConfig, error)
}

var Profiles = []Profile{
	&FullProfile{},
	&FullSafeProfile{},
}

var ProfileNames []string

func ProfileFromString(name string) Profile {
	for _, profile := range Profiles {
		if profile.GetName() == name {
			return profile
		}
	}
	return nil
}

type FullProfile struct{}

func (p *FullProfile) GetName() string {
	return "full"
}
func (p *FullProfile) GetDescription() string {
	return "Complete profile with all tools and extended outputs"
}
func (p *FullProfile) GetTools(s *Server) []server.ServerTool {
	return slices.Concat(
		s.initConfiguration(),
		s.initEvents(),
		s.initNamespaces(),
		s.initPods(),
		s.initResources(),
		s.initHelm(),
	)
}
func (p *FullProfile) GetDefaultConfig() (*config.StaticConfig, error) {
	data, err := fs.ReadFile(defaultFullConfigFile, "configs/full.toml")
	if err != nil {
		return nil, err
	}

	var cfg *config.StaticConfig
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

type FullSafeProfile struct{}

func (p *FullSafeProfile) GetName() string {
	return "full-safe"
}
func (p *FullSafeProfile) GetDescription() string {
	return "Complete profile with all tools and extended outputs"
}
func (p *FullSafeProfile) GetTools(s *Server) []server.ServerTool {
	return slices.Concat(
		s.initEvents(),
		s.initNamespaces(),
		s.initPods(),
		s.initResources(),
		s.initHelm(),
	)
}
func (p *FullSafeProfile) GetDefaultConfig() (*config.StaticConfig, error) {
	data, err := fs.ReadFile(defaultFullSafeConfigFile, "configs/full-safe.toml")
	if err != nil {
		return nil, err
	}

	var cfg *config.StaticConfig
	err = toml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func init() {
	ProfileNames = make([]string, 0)
	for _, profile := range Profiles {
		ProfileNames = append(ProfileNames, profile.GetName())
	}
}
