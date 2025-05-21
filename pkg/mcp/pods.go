package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func (s *Server) initPods() []server.ServerTool {
	return []server.ServerTool{
		{Tool: mcp.NewTool("pods_list",
			mcp.WithDescription("List all the Kubernetes pods in the current cluster from all namespaces"),
			mcp.WithString("labelSelector", mcp.Description("Optional Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the pods by label"), mcp.Pattern("([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]")),
			// Tool annotations
			mcp.WithTitleAnnotation("Pods: List"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.podsListInAllNamespaces},
		{Tool: mcp.NewTool("pods_list_in_namespace",
			mcp.WithDescription("List all the Kubernetes pods in the specified namespace in the current cluster"),
			mcp.WithString("namespace", mcp.Description("Namespace to list pods from"), mcp.Required()),
			mcp.WithString("labelSelector", mcp.Description("Optional Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the pods by label"), mcp.Pattern("([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]")),
			// Tool annotations
			mcp.WithTitleAnnotation("Pods: List in Namespace"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.podsListInNamespace},
		{Tool: mcp.NewTool("pods_get",
			mcp.WithDescription("Get a Kubernetes Pod in the current or provided namespace with the provided name"),
			mcp.WithString("namespace", mcp.Description("Namespace to get the Pod from")),
			mcp.WithString("name", mcp.Description("Name of the Pod"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("Pods: Get"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.podsGet},
		{Tool: mcp.NewTool("pods_delete",
			mcp.WithDescription("Delete a Kubernetes Pod in the current or provided namespace with the provided name"),
			mcp.WithString("namespace", mcp.Description("Namespace to delete the Pod from")),
			mcp.WithString("name", mcp.Description("Name of the Pod to delete"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("Pods: Delete"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.podsDelete},
		{Tool: mcp.NewTool("pods_exec",
			mcp.WithDescription("Execute a command in a Kubernetes Pod in the current or provided namespace with the provided name and command"),
			mcp.WithString("namespace", mcp.Description("Namespace of the Pod where the command will be executed")),
			mcp.WithString("name", mcp.Description("Name of the Pod where the command will be executed"), mcp.Required()),
			mcp.WithArray("command", mcp.Description("Command to execute in the Pod container. "+
				"The first item is the command to be run, and the rest are the arguments to that command. "+
				`Example: ["ls", "-l", "/tmp"]`),
				// TODO: manual fix to ensure that the items property gets initialized (Gemini)
				// https://www.googlecloudcommunity.com/gc/AI-ML/Gemini-API-400-Bad-Request-Array-fields-breaks-function-calling/m-p/769835?nobounce
				func(schema map[string]interface{}) {
					schema["type"] = "array"
					schema["items"] = map[string]interface{}{
						"type": "string",
					}
				},
				mcp.Required(),
			),
			mcp.WithString("container", mcp.Description("Name of the Pod container where the command will be executed (Optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("Pods: Exec"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true), // Depending on the Pod's entrypoint, executing certain commands may kill the Pod
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.podsExec},
		{Tool: mcp.NewTool("pods_log",
			mcp.WithDescription("Get the logs of a Kubernetes Pod in the current or provided namespace with the provided name"),
			mcp.WithString("namespace", mcp.Description("Namespace to get the Pod logs from")),
			mcp.WithString("name", mcp.Description("Name of the Pod to get the logs from"), mcp.Required()),
			mcp.WithString("container", mcp.Description("Name of the Pod container to get the logs from (Optional)")),
			// Tool annotations
			mcp.WithTitleAnnotation("Pods: Log"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.podsLog},
		{Tool: mcp.NewTool("pods_run",
			mcp.WithDescription("Run a Kubernetes Pod in the current or provided namespace with the provided container image and optional name"),
			mcp.WithString("namespace", mcp.Description("Namespace to run the Pod in")),
			mcp.WithString("name", mcp.Description("Name of the Pod (Optional, random name if not provided)")),
			mcp.WithString("image", mcp.Description("Container Image to run in the Pod"), mcp.Required()),
			mcp.WithNumber("port", mcp.Description("TCP/IP port to expose from the Pod container (Optional, no port exposed if not provided)")),
			// Tool annotations
			mcp.WithTitleAnnotation("Pods: Run"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithIdempotentHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.podsRun},
	}
}

func (s *Server) podsListInAllNamespaces(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	labelSelector := ctr.Params.Arguments["labelSelector"]
	var selector string
	if labelSelector != nil {
		selector = labelSelector.(string)
	}

	ret, err := s.k.PodsListInAllNamespaces(ctx, selector)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list pods in all namespaces: %v", err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) podsListInNamespace(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := ctr.Params.Arguments["namespace"]
	if ns == nil {
		return NewTextResult("", errors.New("failed to list pods in namespace, missing argument namespace")), nil
	}
	labelSelector := ctr.Params.Arguments["labelSelector"]
	var selector string
	if labelSelector != nil {
		selector = labelSelector.(string)
	}
	ret, err := s.k.PodsListInNamespace(ctx, ns.(string), selector)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list pods in namespace %s: %v", ns, err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) podsGet(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := ctr.Params.Arguments["namespace"]
	if ns == nil {
		ns = ""
	}
	name := ctr.Params.Arguments["name"]
	if name == nil {
		return NewTextResult("", errors.New("failed to get pod, missing argument name")), nil
	}
	ret, err := s.k.PodsGet(ctx, ns.(string), name.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get pod %s in namespace %s: %v", name, ns, err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) podsDelete(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := ctr.Params.Arguments["namespace"]
	if ns == nil {
		ns = ""
	}
	name := ctr.Params.Arguments["name"]
	if name == nil {
		return NewTextResult("", errors.New("failed to delete pod, missing argument name")), nil
	}
	ret, err := s.k.PodsDelete(ctx, ns.(string), name.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to delete pod %s in namespace %s: %v", name, ns, err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) podsExec(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := ctr.Params.Arguments["namespace"]
	if ns == nil {
		ns = ""
	}
	name := ctr.Params.Arguments["name"]
	if name == nil {
		return NewTextResult("", errors.New("failed to exec in pod, missing argument name")), nil
	}
	container := ctr.Params.Arguments["container"]
	if container == nil {
		container = ""
	}
	commandArg := ctr.Params.Arguments["command"]
	command := make([]string, 0)
	if _, ok := commandArg.([]interface{}); ok {
		for _, cmd := range commandArg.([]interface{}) {
			if _, ok := cmd.(string); ok {
				command = append(command, cmd.(string))
			}
		}
	} else {
		return NewTextResult("", errors.New("failed to exec in pod, invalid command argument")), nil
	}
	ret, err := s.k.PodsExec(ctx, ns.(string), name.(string), container.(string), command)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to exec in pod %s in namespace %s: %v", name, ns, err)), nil
	} else if ret == "" {
		ret = fmt.Sprintf("The executed command in pod %s in namespace %s has not produced any output", name, ns)
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) podsLog(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := ctr.Params.Arguments["namespace"]
	if ns == nil {
		ns = ""
	}
	name := ctr.Params.Arguments["name"]
	if name == nil {
		return NewTextResult("", errors.New("failed to get pod log, missing argument name")), nil
	}
	container := ctr.Params.Arguments["container"]
	if container == nil {
		container = ""
	}
	ret, err := s.k.PodsLog(ctx, ns.(string), name.(string), container.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get pod %s log in namespace %s: %v", name, ns, err)), nil
	} else if ret == "" {
		ret = fmt.Sprintf("The pod %s in namespace %s has not logged any message yet", name, ns)
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) podsRun(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ns := ctr.Params.Arguments["namespace"]
	if ns == nil {
		ns = ""
	}
	name := ctr.Params.Arguments["name"]
	if name == nil {
		name = ""
	}
	image := ctr.Params.Arguments["image"]
	if image == nil {
		return NewTextResult("", errors.New("failed to run pod, missing argument image")), nil
	}
	port := ctr.Params.Arguments["port"]
	if port == nil {
		port = float64(0)
	}
	ret, err := s.k.PodsRun(ctx, ns.(string), name.(string), image.(string), int32(port.(float64)))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get pod %s log in namespace %s: %v", name, ns, err)), nil
	}
	return NewTextResult(ret, err), nil
}
