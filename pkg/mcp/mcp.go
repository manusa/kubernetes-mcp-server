package mcp

import (
	"context"
	"net/http"
	"slices"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/utils/ptr"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
	"github.com/manusa/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
	"github.com/manusa/kubernetes-mcp-server/pkg/version"
)

type Configuration struct {
	Profile    Profile
	ListOutput output.Output

	StaticConfig *config.StaticConfig
}

func (c *Configuration) isToolApplicable(tool server.ServerTool) bool {
	if c.StaticConfig.ReadOnly && !ptr.Deref(tool.Tool.Annotations.ReadOnlyHint, false) {
		return false
	}
	if c.StaticConfig.DisableDestructive && !ptr.Deref(tool.Tool.Annotations.ReadOnlyHint, false) && ptr.Deref(tool.Tool.Annotations.DestructiveHint, false) {
		return false
	}
	if c.StaticConfig.EnabledTools != nil && !slices.Contains(c.StaticConfig.EnabledTools, tool.Tool.Name) {
		return false
	}
	if c.StaticConfig.DisabledTools != nil && slices.Contains(c.StaticConfig.DisabledTools, tool.Tool.Name) {
		return false
	}
	return true
}

type Server struct {
	configuration *Configuration
	server        *server.MCPServer
	k             *kubernetes.Manager
}

func NewServer(configuration Configuration) (*Server, error) {
	s := &Server{
		configuration: &configuration,
		server: server.NewMCPServer(
			version.BinaryName,
			version.Version,
			server.WithResourceCapabilities(true, true),
			server.WithPromptCapabilities(true),
			server.WithToolCapabilities(true),
			server.WithLogging(),
		),
	}
	if err := s.reloadKubernetesClient(); err != nil {
		return nil, err
	}
	s.k.WatchKubeConfig(s.reloadKubernetesClient)

	return s, nil
}

func (s *Server) reloadKubernetesClient() error {
	k, err := kubernetes.NewManager(s.configuration.StaticConfig.KubeConfig, s.configuration.StaticConfig)
	if err != nil {
		return err
	}
	s.k = k
	applicableTools := make([]server.ServerTool, 0)
	for _, tool := range s.configuration.Profile.GetTools(s) {
		if !s.configuration.isToolApplicable(tool) {
			continue
		}
		applicableTools = append(applicableTools, tool)
	}
	s.server.SetTools(applicableTools...)
	return nil
}

func (s *Server) ServeStdio() error {
	return server.ServeStdio(s.server)
}

func (s *Server) ServeSse(baseUrl string, httpServer *http.Server) *server.SSEServer {
	options := make([]server.SSEOption, 0)
	options = append(options, server.WithSSEContextFunc(contextFunc), server.WithHTTPServer(httpServer))
	if baseUrl != "" {
		options = append(options, server.WithBaseURL(baseUrl))
	}
	return server.NewSSEServer(s.server, options...)
}

func (s *Server) ServeHTTP(httpServer *http.Server) *server.StreamableHTTPServer {
	options := []server.StreamableHTTPOption{
		server.WithHTTPContextFunc(contextFunc),
		server.WithStreamableHTTPServer(httpServer),
		server.WithStateLess(true),
	}
	return server.NewStreamableHTTPServer(s.server, options...)
}

func (s *Server) Close() {
	if s.k != nil {
		s.k.Close()
	}
}

func NewTextResult(content string, err error) *mcp.CallToolResult {
	if err != nil {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: err.Error(),
				},
			},
		}
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: content,
			},
		},
	}
}

func contextFunc(ctx context.Context, r *http.Request) context.Context {
	return context.WithValue(ctx, kubernetes.AuthorizationHeader, r.Header.Get(kubernetes.AuthorizationHeader))
}
