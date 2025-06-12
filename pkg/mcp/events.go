package mcp

import (
	"context"
	"fmt"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Server) initEvents() []server.ServerTool {
	return []server.ServerTool{
		{mcp.NewTool("events_list",
			mcp.WithDescription("List all the Kubernetes events in the current cluster from all namespaces"),
			mcp.WithString("namespace",
				mcp.Description("Optional Namespace to retrieve the events from. If not provided, will list events from all namespaces")),
			// Tool annotations
			mcp.WithTitleAnnotation("Events: List"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), s.eventsList},
	}
}

func (s *Server) eventsList(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace := ctr.GetArguments()["namespace"]
	if namespace == nil {
		namespace = ""
	}
	eventMap, err := s.k.Derived(ctx).EventsList(ctx, namespace.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list events in all namespaces: %v", err)), nil
	}
	if len(eventMap) == 0 {
		return NewTextResult("No events found", nil), nil
	}
	yamlEvents, err := output.MarshalYaml(eventMap)
	if err != nil {
		err = fmt.Errorf("failed to list events in all namespaces: %v", err)
	}
	return NewTextResult(fmt.Sprintf("The following events (YAML format) were found:\n%s", yamlEvents), err), nil
}
