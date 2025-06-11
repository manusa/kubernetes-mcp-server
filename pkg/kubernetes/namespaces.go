package kubernetes

import (
	"context"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (k *Kubernetes) NamespacesList(ctx context.Context) ([]unstructured.Unstructured, error) {
	return k.ResourcesList(ctx, &schema.GroupVersionKind{
		Group: "", Version: "v1", Kind: "Namespace",
	}, "")
}

func (k *Kubernetes) ProjectsList(ctx context.Context) ([]unstructured.Unstructured, error) {
	return k.ResourcesList(ctx, &schema.GroupVersionKind{
		Group: "project.openshift.io", Version: "v1", Kind: "Project",
	}, "")
}
