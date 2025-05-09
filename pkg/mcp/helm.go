package mcp

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Server) initHelm() []server.ServerTool {
	rets := make([]server.ServerTool, 0)
	rets = append(rets, server.ServerTool{
		Tool: mcp.NewTool("helm_list",
			mcp.WithDescription("List all of the Helm releases in the current or provided namespace (or in all namespaces if specified)"),
			mcp.WithString("namespace", mcp.Description("Namespace to list Helm releases from (Optional, all namespaces if not provided)")),
			mcp.WithBoolean("all_namespaces", mcp.Description("If true, lists all Helm releases in all namespaces ignoring the namespace argument (Optional)")),
		),
		Handler: s.helmList,
	})
	return rets
}

func (s *Server) helmList(_ context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	allNamespaces := false
	if v, ok := ctr.Params.Arguments["all_namespaces"].(bool); ok {
		allNamespaces = v
	}
	namespace := ""
	if v, ok := ctr.Params.Arguments["namespace"].(string); ok {
		namespace = v
	}
	ret, err := s.k.Helm.ReleasesList(namespace, allNamespaces)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list helm releases in namespace '%s': %w", namespace, err)), nil
	}
	return NewTextResult(ret, err), nil
}
