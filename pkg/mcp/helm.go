package mcp

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"gopkg.in/yaml.v3"
)

func (s *Server) initHelm() []server.ServerTool {
	rets := make([]server.ServerTool, 0)
	rets = append(rets, server.ServerTool{
		Tool: mcp.NewTool("helm_list",
			mcp.WithDescription("List all Helm releases in all namespaces."),
		),
		Handler: s.helmReleasesList,
	})
	rets = append(rets, server.ServerTool{
		Tool: mcp.NewTool("helm_list_in_namespace",
			mcp.WithDescription("List all Helm releases in the specified namespace."),
			mcp.WithString("namespace", mcp.Description("Namespace to list Helm releases from."), mcp.Required()),
		),
		Handler: s.helmListInNamespace,
	})
	return rets
}

func (s *Server) helmReleasesList(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	releases, err := s.helm.ReleasesList(ctx, "")
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list helm releases: %w", err)), nil
	}
	for _, r := range releases {
		if r != nil && r.Chart != nil {
			r.Chart.Templates = nil
			r.Chart.Files = nil
		}
	}
	out, err := yaml.Marshal(releases)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to marshal helm releases: %w", err)), nil
	}
	return NewTextResult(string(out), nil), nil
}

// helmListInNamespace lists Helm releases in a specified namespace
func (s *Server) helmListInNamespace(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := req.Params.Arguments["namespace"]
	if ns == nil || ns.(string) == "" {
		return NewTextResult("", fmt.Errorf("missing required argument: namespace")), nil
	}
	releases, err := s.helm.ReleasesList(ctx, ns.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list helm releases in namespace %s: %w", ns, err)), nil
	}
	for _, r := range releases {
		if r != nil && r.Chart != nil {
			r.Chart.Templates = nil
			r.Chart.Files = nil
		}
	}
	out, err := yaml.Marshal(releases)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to marshal helm releases: %w", err)), nil
	}
	return NewTextResult(string(out), nil), nil
}
