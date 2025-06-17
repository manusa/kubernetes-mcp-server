package kubernetes

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func (k *kubernetes) IsOpenShift(ctx context.Context) bool {
	// This method should be fast and not block (it's called at startup)
	timeoutSeconds := int64(5)
	if _, err := k.dynamicClient.Resource(schema.GroupVersionResource{
		Group:    "project.openshift.io",
		Version:  "v1",
		Resource: "projects",
	}).List(ctx, metav1.ListOptions{Limit: 1, TimeoutSeconds: &timeoutSeconds}); err == nil {
		return true
	}
	return false
}
