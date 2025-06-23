package kubernetes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/restmapper"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
)

type AccessControlRESTMapper struct {
	delegate     *restmapper.DeferredDiscoveryRESTMapper
	staticConfig *config.StaticConfig // TODO: maybe just store the denied resource slice
}

var _ meta.RESTMapper = &AccessControlRESTMapper{}

// isAllowed checks the resource is in denied list or not.
// If it is in denied list, this function returns false.
func (a AccessControlRESTMapper) isAllowed(gvk *schema.GroupVersionKind) bool {
	if a.staticConfig == nil {
		return true
	}

	for _, val := range a.staticConfig.DeniedResources {
		// If kind is empty, that means Group/Version pair is denied entirely
		if val.Kind == "" {
			if gvk.Group == val.Group && gvk.Version == val.Version {
				return false
			}
		}
		if gvk.Group == val.Group &&
			gvk.Version == val.Version &&
			gvk.Kind == val.Kind {
			return false
		}
	}

	return true
}

func (a AccessControlRESTMapper) KindFor(resource schema.GroupVersionResource) (schema.GroupVersionKind, error) {
	gvk, err := a.delegate.KindFor(resource)
	if err != nil {
		return schema.GroupVersionKind{}, err
	}
	if !a.isAllowed(&gvk) {
		return schema.GroupVersionKind{}, fmt.Errorf("resource not allowed: %s", gvk.String())
	}
	return gvk, nil
}

func (a AccessControlRESTMapper) KindsFor(resource schema.GroupVersionResource) ([]schema.GroupVersionKind, error) {
	gvks, err := a.delegate.KindsFor(resource)
	if err != nil {
		return nil, err
	}
	for i := range gvks {
		if !a.isAllowed(&gvks[i]) {
			return nil, fmt.Errorf("resource not allowed: %s", gvks[i].String())
		}
	}
	return gvks, nil
}

func (a AccessControlRESTMapper) ResourceFor(input schema.GroupVersionResource) (schema.GroupVersionResource, error) {
	return a.delegate.ResourceFor(input)
}

func (a AccessControlRESTMapper) ResourcesFor(input schema.GroupVersionResource) ([]schema.GroupVersionResource, error) {
	return a.delegate.ResourcesFor(input)
}

func (a AccessControlRESTMapper) RESTMapping(gk schema.GroupKind, versions ...string) (*meta.RESTMapping, error) {
	for _, version := range versions {
		gvk := &schema.GroupVersionKind{Group: gk.Group, Version: version, Kind: gk.Kind}
		if !a.isAllowed(gvk) {
			return nil, fmt.Errorf("resource not allowed: %s", gvk.String())
		}
	}
	return a.delegate.RESTMapping(gk, versions...)
}

func (a AccessControlRESTMapper) RESTMappings(gk schema.GroupKind, versions ...string) ([]*meta.RESTMapping, error) {
	for _, version := range versions {
		gvk := &schema.GroupVersionKind{Group: gk.Group, Version: version, Kind: gk.Kind}
		if !a.isAllowed(gvk) {
			return nil, fmt.Errorf("resource not allowed: %s", gvk.String())
		}
	}
	return a.delegate.RESTMappings(gk, versions...)
}

func (a AccessControlRESTMapper) ResourceSingularizer(resource string) (singular string, err error) {
	return a.delegate.ResourceSingularizer(resource)
}

func (a AccessControlRESTMapper) Reset() {
	a.delegate.Reset()
}

func NewAccessControlRESTMapper(delegate *restmapper.DeferredDiscoveryRESTMapper, staticConfig *config.StaticConfig) *AccessControlRESTMapper {
	return &AccessControlRESTMapper{delegate: delegate, staticConfig: staticConfig}
}
