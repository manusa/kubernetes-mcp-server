package mcp

import (
	"context"
	"errors"
	"fmt"
	"github.com/manusa/kubernetes-mcp-server/pkg/kubernetes"
	"github.com/mark3labs/mcp-go/mcp"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (s *Sever) initResources() {
	s.server.AddTool(mcp.NewTool(
		"resources_list",
		mcp.WithDescription("List Kubernetes resources in the current cluster by providing their apiVersion and kind and optionally the namespace"),
		mcp.WithString("apiVersion",
			mcp.Description("apiVersion of the resources (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)"),
			mcp.Required(),
		),
		mcp.WithString("kind",
			mcp.Description("kind of the resources (examples of valid kind are: Pod, Service, Deployment, Ingress)"),
			mcp.Required(),
		),
		mcp.WithString("namespace",
			mcp.Description("Optional Namespace to retrieve the namespaced resources from (ignored in case of cluster scoped resources). If not provided, will list resources from all namespaces"),
		),
	), resourcesList)
	s.server.AddTool(mcp.NewTool(
		"resources_get",
		mcp.WithDescription("Get a Kubernetes resource in the current cluster by providing its apiVersion, kind, optionally the namespace, and its name"),
		mcp.WithString("apiVersion",
			mcp.Description("apiVersion of the resource (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)"),
			mcp.Required(),
		),
		mcp.WithString("kind",
			mcp.Description("kind of the resource (examples of valid kind are: Pod, Service, Deployment, Ingress)"),
			mcp.Required(),
		),
		mcp.WithString("namespace",
			mcp.Description("Optional Namespace to retrieve the namespaced resource from (ignored in case of cluster scoped resources). If not provided, will get resource from configured namespace"),
		),
		mcp.WithString("name",
			mcp.Description("Name of the resource"),
			mcp.Required(),
		),
	), resourcesGet)
	s.server.AddTool(mcp.NewTool(
		"resources_create_or_update",
		mcp.WithDescription("Create or update a Kubernetes resource in the current cluster by providing a YAML or JSON representation of the resource"),
		mcp.WithString("resource",
			mcp.Description("A JSON or YAML containing a representation of the Kubernetes resource. Should include top-level fields such as apiVersion,kind,metadata, and spec"),
			mcp.Required(),
		),
	), resourcesCreateOrUpdate)
}

func resourcesList(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	k, err := kubernetes.NewKubernetes()
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list resources: %v", err)), nil
	}
	namespace := ctr.Params.Arguments["namespace"]
	if namespace == nil {
		namespace = ""
	}
	gvk, err := parseGroupVersionKind(ctr.Params.Arguments)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list resources, %s", err)), nil
	}
	ret, err := k.ResourcesList(ctx, gvk, namespace.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list resources: %v", err)), nil
	}
	return NewTextResult(ret, err), nil
}

func resourcesGet(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	k, err := kubernetes.NewKubernetes()
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get resource: %v", err)), nil
	}
	namespace := ctr.Params.Arguments["namespace"]
	if namespace == nil {
		namespace = ""
	}
	gvk, err := parseGroupVersionKind(ctr.Params.Arguments)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get resource, %s", err)), nil
	}
	name := ctr.Params.Arguments["name"]
	if name == nil {
		return NewTextResult("", errors.New("failed to get resource, missing argument name")), nil
	}
	ret, err := k.ResourcesGet(ctx, gvk, namespace.(string), name.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get resource: %v", err)), nil
	}
	return NewTextResult(ret, err), nil
}

func resourcesCreateOrUpdate(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	k, err := kubernetes.NewKubernetes()
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to create or update resources: %v", err)), nil
	}
	resource := ctr.Params.Arguments["resource"]
	if resource == nil || resource == "" {
		return NewTextResult("", errors.New("failed to create or update resources, missing argument resource")), nil
	}
	ret, err := k.ResourcesCreateOrUpdate(ctx, resource.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to create or update resources: %v", err)), nil
	}
	return NewTextResult(ret, err), nil
}

func parseGroupVersionKind(arguments map[string]interface{}) (*schema.GroupVersionKind, error) {
	apiVersion := arguments["apiVersion"]
	if apiVersion == nil {
		return nil, errors.New("missing argument apiVersion")
	}
	kind := arguments["kind"]
	if kind == nil {
		return nil, errors.New("missing argument kind")
	}
	gv, err := schema.ParseGroupVersion(apiVersion.(string))
	if err != nil {
		return nil, errors.New("invalid argument apiVersion")
	}
	return &schema.GroupVersionKind{Group: gv.Group, Version: gv.Version, Kind: kind.(string)}, nil
}
