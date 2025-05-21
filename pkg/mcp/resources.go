package mcp

import (
	"context"
	"errors"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (s *Server) initResources() []server.ServerTool {
	commonApiVersion := "v1 Pod, v1 Service, v1 Node, apps/v1 Deployment, networking.k8s.io/v1 Ingress"
	if s.k.IsOpenShift(context.Background()) {
		commonApiVersion += ", route.openshift.io/v1 Route"
	}
	commonApiVersion = fmt.Sprintf("(common apiVersion and kind include: %s)", commonApiVersion)
	return []server.ServerTool{
		{Tool: mcp.NewTool("resources_list",
			mcp.WithDescription("List Kubernetes resources and objects in the current cluster by providing their apiVersion and kind and optionally the namespace and label selector\n"+
				commonApiVersion),
			mcp.WithString("apiVersion",
				mcp.Description("apiVersion of the resources (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)"),
				mcp.Required(),
			),
			mcp.WithString("kind",
				mcp.Description("kind of the resources (examples of valid kind are: Pod, Service, Deployment, Ingress)"),
				mcp.Required(),
			),
			mcp.WithString("namespace",
				mcp.Description("Optional Namespace to retrieve the namespaced resources from (ignored in case of cluster scoped resources). If not provided, will list resources from all namespaces")),
			mcp.WithString("labelSelector",
				mcp.Description("Optional Kubernetes label selector (e.g. 'app=myapp,env=prod' or 'app in (myapp,yourapp)'), use this option when you want to filter the pods by label"), mcp.Pattern("([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]")),
			// Tool annotations
			mcp.WithTitleAnnotation("Resources: List"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.resourcesList},
		{Tool: mcp.NewTool("resources_get",
			mcp.WithDescription("Get a Kubernetes resource in the current cluster by providing its apiVersion, kind, optionally the namespace, and its name\n"+
				commonApiVersion),
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
			mcp.WithString("name", mcp.Description("Name of the resource"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("Resources: Get"),
			mcp.WithReadOnlyHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.resourcesGet},
		{Tool: mcp.NewTool("resources_create_or_update",
			mcp.WithDescription("Create or update a Kubernetes resource in the current cluster by providing a YAML or JSON representation of the resource\n"+
				commonApiVersion),
			mcp.WithString("resource",
				mcp.Description("A JSON or YAML containing a representation of the Kubernetes resource. Should include top-level fields such as apiVersion,kind,metadata, and spec"),
				mcp.Required(),
			),
			// Tool annotations
			mcp.WithTitleAnnotation("Resources: Create or Update"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.resourcesCreateOrUpdate},
		{Tool: mcp.NewTool("resources_delete",
			mcp.WithDescription("Delete a Kubernetes resource in the current cluster by providing its apiVersion, kind, optionally the namespace, and its name\n"+
				commonApiVersion),
			mcp.WithString("apiVersion",
				mcp.Description("apiVersion of the resource (examples of valid apiVersion are: v1, apps/v1, networking.k8s.io/v1)"),
				mcp.Required(),
			),
			mcp.WithString("kind",
				mcp.Description("kind of the resource (examples of valid kind are: Pod, Service, Deployment, Ingress)"),
				mcp.Required(),
			),
			mcp.WithString("namespace",
				mcp.Description("Optional Namespace to delete the namespaced resource from (ignored in case of cluster scoped resources). If not provided, will delete resource from configured namespace"),
			),
			mcp.WithString("name", mcp.Description("Name of the resource"), mcp.Required()),
			// Tool annotations
			mcp.WithTitleAnnotation("Resources: Delete"),
			mcp.WithReadOnlyHintAnnotation(false),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithIdempotentHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		), Handler: s.resourcesDelete},
	}
}

func (s *Server) resourcesList(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace := ctr.Params.Arguments["namespace"]
	if namespace == nil {
		namespace = ""
	}
	labelSelector := ctr.Params.Arguments["labelSelector"]
	if labelSelector == nil {
		labelSelector = ""
	}
	gvk, err := parseGroupVersionKind(ctr.Params.Arguments)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list resources, %s", err)), nil
	}
	ret, err := s.k.ResourcesList(ctx, gvk, namespace.(string), labelSelector.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to list resources: %v", err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) resourcesGet(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
	ret, err := s.k.ResourcesGet(ctx, gvk, namespace.(string), name.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to get resource: %v", err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) resourcesCreateOrUpdate(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resource := ctr.Params.Arguments["resource"]
	if resource == nil || resource == "" {
		return NewTextResult("", errors.New("failed to create or update resources, missing argument resource")), nil
	}
	ret, err := s.k.ResourcesCreateOrUpdate(ctx, resource.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to create or update resources: %v", err)), nil
	}
	return NewTextResult(ret, err), nil
}

func (s *Server) resourcesDelete(ctx context.Context, ctr mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	namespace := ctr.Params.Arguments["namespace"]
	if namespace == nil {
		namespace = ""
	}
	gvk, err := parseGroupVersionKind(ctr.Params.Arguments)
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to delete resource, %s", err)), nil
	}
	name := ctr.Params.Arguments["name"]
	if name == nil {
		return NewTextResult("", errors.New("failed to delete resource, missing argument name")), nil
	}
	err = s.k.ResourcesDelete(ctx, gvk, namespace.(string), name.(string))
	if err != nil {
		return NewTextResult("", fmt.Errorf("failed to delete resource: %v", err)), nil
	}
	return NewTextResult("Resource deleted successfully", err), nil
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
