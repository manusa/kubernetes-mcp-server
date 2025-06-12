package mcp

import (
	"context"
	"fmt"
	"github.com/manusa/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Server) initNamespaces() []server.ServerTool {
	ret := make([]server.ServerTool, 0)
	ret = append(ret, server.ServerTool{
		Tool: mcp.NewTool("namespaces_list",
			mcp.WithDescription("List all the Kubernetes namespaces in the current cluster"),
			// Tool annotations
			mcp.WithTitleAnnotation("Namespaces: List"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.namespacesList,
	})
	if s.k.IsOpenShift(context.Background()) {
		ret = append(ret, server.ServerTool{
			Tool: mcp.NewTool("projects_list",
				mcp.WithDescription("List all the OpenShift projects in the current cluster"),
				// Tool annotations
				mcp.WithTitleAnnotation("Projects: List"),
				mcp.WithReadOnlyHintAnnotation(true),
				mcp.WithDestructiveHintAnnotation(false),
				mcp.WithOpenWorldHintAnnotation(true),
			), Handler: s.projectsList,
		})
	}
	return ret
}

func (s *Server) namespacesList(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ret, err := s.k.Derived(ctx).NamespacesList(ctx, kubernetes.ResourceListOptions{AsTable: s.configuration.ListOutput.AsTable()})
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list namespaces: %v", err)), nil
	}
	return NewTextResult(s.configuration.ListOutput.PrintObj(ret)), nil
}

func (s *Server) projectsList(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ret, err := s.k.Derived(ctx).ProjectsList(ctx, kubernetes.ResourceListOptions{AsTable: s.configuration.ListOutput.AsTable()})
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list projects: %v", err)), nil
	}
	return NewTextResult(s.configuration.ListOutput.PrintObj(ret)), nil
}
