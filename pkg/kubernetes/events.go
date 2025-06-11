package kubernetes

import (
	"context"
	"fmt"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"strings"
)

func (k *Kubernetes) EventsList(ctx context.Context, namespace string) (string, error) {
	unstructuredList, err := k.resourcesList(ctx, &schema.GroupVersionKind{
		Group: "", Version: "v1", Kind: "Event",
	}, namespace, "")
	if err != nil {
		return "", err
	}
	if len(unstructuredList.Items) == 0 {
		return "No events found", nil
	}
	var eventMap []map[string]any
	for _, item := range unstructuredList.Items {
		event := &v1.Event{}
		if err = runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, event); err != nil {
			return "", err
		}
		timestamp := event.EventTime.Time
		if timestamp.IsZero() && event.Series != nil {
			timestamp = event.Series.LastObservedTime.Time
		} else if timestamp.IsZero() && event.Count > 1 {
			timestamp = event.LastTimestamp.Time
		} else if timestamp.IsZero() {
			timestamp = event.FirstTimestamp.Time
		}
		eventMap = append(eventMap, map[string]any{
			"Namespace": event.Namespace,
			"Timestamp": timestamp.String(),
			"Type":      event.Type,
			"Reason":    event.Reason,
			"InvolvedObject": map[string]string{
				"apiVersion": event.InvolvedObject.APIVersion,
				"Kind":       event.InvolvedObject.Kind,
				"Name":       event.InvolvedObject.Name,
			},
			"Message": strings.TrimSpace(event.Message),
		})
	}
	yamlEvents, err := output.MarshalYaml(eventMap)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("The following events (YAML format) were found:\n%s", yamlEvents), nil
}
