package kubernetes

import (
	"context"
	"regexp"
	"strings"

	"github.com/manusa/kubernetes-mcp-server/pkg/version"
	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

const (
	AppKubernetesComponent = "app.kubernetes.io/component"
	AppKubernetesManagedBy = "app.kubernetes.io/managed-by"
	AppKubernetesName      = "app.kubernetes.io/name"
	AppKubernetesPartOf    = "app.kubernetes.io/part-of"
)

func (k *Kubernetes) ResourcesList(ctx context.Context, gvk *schema.GroupVersionKind, namespace string, labelSelector ...string) (string, error) {
	var selector string
	if len(labelSelector) > 0 {
		selector = labelSelector[0]
	}
	rl, err := k.resourcesList(ctx, gvk, namespace, selector)
	if err != nil {
		return "", err
	}
	return marshal(rl.Items)
}

func (k *Kubernetes) ResourcesGet(ctx context.Context, gvk *schema.GroupVersionKind, namespace, name string) (string, error) {
	gvr, err := k.resourceFor(gvk)
	if err != nil {
		return "", err
	}
	// If it's a namespaced resource and namespace wasn't provided, try to use the default configured one
	if namespaced, nsErr := k.isNamespaced(gvk); nsErr == nil && namespaced {
		namespace = k.NamespaceOrDefault(namespace)
	}
	rg, err := k.dynamicClient.Resource(*gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return marshal(rg)
}

func (k *Kubernetes) ResourcesCreateOrUpdate(ctx context.Context, resource string) (string, error) {
	separator := regexp.MustCompile(`\r?\n---\r?\n`)
	resources := separator.Split(resource, -1)
	var parsedResources []*unstructured.Unstructured
	for _, r := range resources {
		var obj unstructured.Unstructured
		if err := yaml.NewYAMLToJSONDecoder(strings.NewReader(r)).Decode(&obj); err != nil {
			return "", err
		}
		parsedResources = append(parsedResources, &obj)
	}
	return k.resourcesCreateOrUpdate(ctx, parsedResources)
}

func (k *Kubernetes) ResourcesDelete(ctx context.Context, gvk *schema.GroupVersionKind, namespace, name string) error {
	gvr, err := k.resourceFor(gvk)
	if err != nil {
		return err
	}
	// If it's a namespaced resource and namespace wasn't provided, try to use the default configured one
	if namespaced, nsErr := k.isNamespaced(gvk); nsErr == nil && namespaced {
		namespace = k.NamespaceOrDefault(namespace)
	}
	return k.dynamicClient.Resource(*gvr).Namespace(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (k *Kubernetes) resourcesList(ctx context.Context, gvk *schema.GroupVersionKind, namespace string, labelSelector string) (*unstructured.UnstructuredList, error) {
	gvr, err := k.resourceFor(gvk)
	if err != nil {
		return nil, err
	}
	// Check if operation is allowed for all namespaces (applicable for namespaced resources)
	isNamespaced, _ := k.isNamespaced(gvk)
	if isNamespaced && !k.canIUse(ctx, gvr, namespace, "list") && namespace == "" {
		namespace = k.configuredNamespace()
	}
	return k.dynamicClient.Resource(*gvr).Namespace(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}

func (k *Kubernetes) resourcesCreateOrUpdate(ctx context.Context, resources []*unstructured.Unstructured) (string, error) {
	for i, obj := range resources {
		gvk := obj.GroupVersionKind()
		gvr, rErr := k.resourceFor(&gvk)
		if rErr != nil {
			return "", rErr
		}
		namespace := obj.GetNamespace()
		// If it's a namespaced resource and namespace wasn't provided, try to use the default configured one
		if namespaced, nsErr := k.isNamespaced(&gvk); nsErr == nil && namespaced {
			namespace = k.NamespaceOrDefault(namespace)
		}
		resources[i], rErr = k.dynamicClient.Resource(*gvr).Namespace(namespace).Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{
			FieldManager: version.BinaryName,
		})
		if rErr != nil {
			return "", rErr
		}
		// Clear the cache to ensure the next operation is performed on the latest exposed APIs
		if gvk.Kind == "CustomResourceDefinition" {
			k.deferredDiscoveryRESTMapper.Reset()
		}
	}
	marshalledYaml, err := marshal(resources)
	if err != nil {
		return "", err
	}
	return "# The following resources (YAML) have been created or updated successfully\n" + marshalledYaml, nil
}

func (k *Kubernetes) resourceFor(gvk *schema.GroupVersionKind) (*schema.GroupVersionResource, error) {
	m, err := k.deferredDiscoveryRESTMapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
	if err != nil {
		return nil, err
	}
	return &m.Resource, nil
}

func (k *Kubernetes) isNamespaced(gvk *schema.GroupVersionKind) (bool, error) {
	apiResourceList, err := k.discoveryClient.ServerResourcesForGroupVersion(gvk.GroupVersion().String())
	if err != nil {
		return false, err
	}
	for _, apiResource := range apiResourceList.APIResources {
		if apiResource.Kind == gvk.Kind {
			return apiResource.Namespaced, nil
		}
	}
	return false, nil
}

func (k *Kubernetes) supportsGroupVersion(groupVersion string) bool {
	if _, err := k.discoveryClient.ServerResourcesForGroupVersion(groupVersion); err != nil {
		return false
	}
	return true
}

func (k *Kubernetes) canIUse(ctx context.Context, gvr *schema.GroupVersionResource, namespace, verb string) bool {
	response, err := k.clientSet.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, &authv1.SelfSubjectAccessReview{
		Spec: authv1.SelfSubjectAccessReviewSpec{ResourceAttributes: &authv1.ResourceAttributes{
			Namespace: namespace,
			Verb:      verb,
			Group:     gvr.Group,
			Version:   gvr.Version,
			Resource:  gvr.Resource,
		}},
	}, metav1.CreateOptions{})
	if err != nil {
		// TODO: maybe return the error too
		return false
	}
	return response.Status.Allowed
}
