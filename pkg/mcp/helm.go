package mcp

import (
	"context"
	"fmt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Server) initHelm() []server.ServerTool {
	return []server.ServerTool{
		{mcp.NewTool("helm_install",
			mcp.WithDescription("Install a Helm chart in the current or provided namespace"),
			mcp.WithString("chart", mcp.Description("Chart reference to install (for example: stable/grafana, oci://ghcr.io/nginxinc/charts/nginx-ingress)"), mcp.Required()),
			mcp.WithObject("values", mcp.Description("Values to pass to the Helm chart (Optional)")),
			mcp.WithString("name", mcp.Description("Name of the Helm release (Optional, random name if not provided)")),
			mcp.WithString("namespace", mcp.Description("Namespace to install the Helm chart in (Optional, current namespace if not provided)")),
		), s.helmInstall},
		{mcp.NewTool("helm_list",
			mcp.WithDescription("List all the Helm releases in the current or provided namespace (or in all namespaces if specified)"),
			mcp.WithString("namespace", mcp.Description("Namespace to list Helm releases from (Optional, all namespaces if not provided)")),
			mcp.WithBoolean("all_namespaces", mcp.Description("If true, lists all Helm releases in all namespaces ignoring the namespace argument (Optional)")),
		), s.helmList},
		{mcp.NewTool("helm_uninstall",
			mcp.WithDescription("Uninstall a Helm release in the current or provided namespace"),
			mcp.WithString("name", mcp.Description("Name of the Helm release to uninstall"), mcp.Required()),
			mcp.WithString("namespace", mcp.Description("Namespace to uninstall the Helm release from (Optional, current namespace if not provided)")),
		), s.helmUninstall},
	}
}

func (s *Server) helmInstall(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var chart string
	ok := false
	if chart, ok = ctr.Params.Arguments["chart"].(string); !ok {
		return NewTextResult("", fmt.Errorf("failed to install helm chart, missing argument chart")), nil
	}
	values := map[string]interface{}{}
	if v, ok := ctr.Params.Arguments["values"].(map[string]interface{}); ok {
		values = v
	}
	name := ""
	if v, ok := ctr.Params.Arguments["name"].(string); ok {
		name = v
	}
	namespace := ""
	if v, ok := ctr.Params.Arguments["namespace"].(string); ok {
		namespace = v
	}
	ret, err := s.k.Helm.Install(ctx, chart, values, name, namespace)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to install helm chart '%s': %w", chart, err)), nil
	}
	return NewTextResult(ret, err), nil
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
	ret, err := s.k.Helm.List(namespace, allNamespaces)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list helm releases in namespace '%s': %w", namespace, err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) helmUninstall(_ context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var name string
	ok := false
	if name, ok = ctr.Params.Arguments["name"].(string); !ok {
		return NewTextResult("", fmt.Errorf("failed to uninstall helm chart, missing argument name")), nil
	}
	namespace := ""
	if v, ok := ctr.Params.Arguments["namespace"].(string); ok {
		namespace = v
	}
	ret, err := s.k.Helm.Uninstall(name, namespace)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to uninstall helm chart '%s': %w", name, err)), nil
	}
	return NewTextResult(ret, err), nil
}
