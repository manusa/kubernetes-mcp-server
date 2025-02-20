package mcp

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"
)

func TestPodsListInAllNamespaces(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		toolResult, err := c.callTool("pods_list", map[string]interface{}{})
		t.Run("pods_list returns pods list", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
				return
			}
			if toolResult.IsError {
				t.Fatalf("call tool failed")
				return
			}
		})
		var decoded []unstructured.Unstructured
		err = yaml.Unmarshal([]byte(toolResult.Content[0].(map[string]interface{})["text"].(string)), &decoded)
		t.Run("pods_list has yaml content", func(t *testing.T) {
			if err != nil {
				t.Fatalf("invalid tool result content %v", err)
				return
			}
		})
		t.Run("pods_list returns 3 items", func(t *testing.T) {
			if len(decoded) != 3 {
				t.Fatalf("invalid pods count, expected 3, got %v", len(decoded))
				return
			}
		})
		t.Run("pods_list returns pod in ns-1", func(t *testing.T) {
			if decoded[1].GetName() != "a-pod-in-ns-1" {
				t.Fatalf("invalid pod name, expected a-pod-in-ns-1, got %v", decoded[1].GetName())
				return
			}
			if decoded[1].GetNamespace() != "ns-1" {
				t.Fatalf("invalid pod namespace, expected ns-1, got %v", decoded[1].GetNamespace())
				return
			}
		})
		t.Run("pods_list returns pod in ns-2", func(t *testing.T) {
			if decoded[2].GetName() != "a-pod-in-ns-2" {
				t.Fatalf("invalid pod name, expected a-pod-in-ns-2, got %v", decoded[2].GetName())
				return
			}
			if decoded[2].GetNamespace() != "ns-2" {
				t.Fatalf("invalid pod namespace, expected ns-2, got %v", decoded[2].GetNamespace())
				return
			}
		})
		t.Run("pods_list omits managed fields", func(t *testing.T) {
			if decoded[1].GetManagedFields() != nil {
				t.Fatalf("managed fields should be omitted, got %v", decoded[0].GetManagedFields())
				return
			}
		})
	})
}

func TestPodsListInAllNamespacesUnauthorized(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		defer restoreAuth(c.ctx)
		client := c.newKubernetesClient()
		// Authorize user only for default/configured namespace
		r, _ := client.RbacV1().Roles("default").Create(c.ctx, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-pods-list"},
			Rules: []rbacv1.PolicyRule{{
				Verbs:     []string{"get", "list"},
				APIGroups: []string{""},
				Resources: []string{"pods"},
			}},
		}, metav1.CreateOptions{})
		_, _ = client.RbacV1().RoleBindings("default").Create(c.ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: "allow-pods-list"},
			Subjects:   []rbacv1.Subject{{Kind: "User", Name: envTestUser.Name}},
			RoleRef:    rbacv1.RoleRef{Kind: "Role", Name: r.Name},
		}, metav1.CreateOptions{})
		// Deny cluster by removing cluster rule
		_ = client.RbacV1().ClusterRoles().Delete(c.ctx, "allow-all", metav1.DeleteOptions{})
		toolResult, err := c.callTool("pods_list", map[string]interface{}{})
		t.Run("pods_list returns pods list for default namespace only", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
				return
			}
			if toolResult.IsError {
				t.Fatalf("call tool failed")
				return
			}
		})
		var decoded []unstructured.Unstructured
		err = yaml.Unmarshal([]byte(toolResult.Content[0].(map[string]interface{})["text"].(string)), &decoded)
		t.Run("pods_list has yaml content", func(t *testing.T) {
			if err != nil {
				t.Fatalf("invalid tool result content %v", err)
				return
			}
		})
		t.Run("pods_list returns 1 items", func(t *testing.T) {
			if len(decoded) != 1 {
				t.Fatalf("invalid pods count, expected 1, got %v", len(decoded))
				return
			}
		})
		t.Run("pods_list returns pod in default", func(t *testing.T) {
			if decoded[0].GetName() != "a-pod-in-default" {
				t.Fatalf("invalid pod name, expected a-pod-in-default, got %v", decoded[0].GetName())
				return
			}
			if decoded[0].GetNamespace() != "default" {
				t.Fatalf("invalid pod namespace, expected default, got %v", decoded[0].GetNamespace())
				return
			}
		})
	})
}

func TestPodsListInNamespace(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		t.Run("pods_list_in_namespace with nil namespace returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_list_in_namespace", map[string]interface{}{})
			if toolResult.IsError != true {
				t.Fatalf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to list pods in namespace, missing argument namespace" {
				t.Fatalf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		toolResult, err := c.callTool("pods_list_in_namespace", map[string]interface{}{
			"namespace": "ns-1",
		})
		t.Run("pods_list_in_namespace returns pods list", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
				return
			}
			if toolResult.IsError {
				t.Fatalf("call tool failed")
				return
			}
		})
		var decoded []unstructured.Unstructured
		err = yaml.Unmarshal([]byte(toolResult.Content[0].(map[string]interface{})["text"].(string)), &decoded)
		t.Run("pods_list_in_namespace has yaml content", func(t *testing.T) {
			if err != nil {
				t.Fatalf("invalid tool result content %v", err)
				return
			}
		})
		t.Run("pods_list_in_namespace returns 1 items", func(t *testing.T) {
			if len(decoded) != 1 {
				t.Fatalf("invalid pods count, expected 1, got %v", len(decoded))
				return
			}
		})
		t.Run("pods_list_in_namespace returns pod in ns-1", func(t *testing.T) {
			if decoded[0].GetName() != "a-pod-in-ns-1" {
				t.Fatalf("invalid pod name, expected a-pod-in-ns-1, got %v", decoded[0].GetName())
				return
			}
			if decoded[0].GetNamespace() != "ns-1" {
				t.Fatalf("invalid pod namespace, expected ns-1, got %v", decoded[0].GetNamespace())
				return
			}
		})
		t.Run("pods_list_in_namespace omits managed fields", func(t *testing.T) {
			if decoded[0].GetManagedFields() != nil {
				t.Fatalf("managed fields should be omitted, got %v", decoded[0].GetManagedFields())
				return
			}
		})
	})
}

func TestPodsGet(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		t.Run("pods_get with nil name returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_get", map[string]interface{}{})
			if toolResult.IsError != true {
				t.Fatalf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to get pod, missing argument name" {
				t.Fatalf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		t.Run("pods_get with not found name returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_get", map[string]interface{}{"name": "not-found"})
			if toolResult.IsError != true {
				t.Fatalf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to get pod not-found in namespace : pods \"not-found\" not found" {
				t.Fatalf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		podsGetNilNamespace, err := c.callTool("pods_get", map[string]interface{}{
			"name": "a-pod-in-default",
		})
		t.Run("pods_get with name and nil namespace returns pod", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
				return
			}
			if podsGetNilNamespace.IsError {
				t.Fatalf("call tool failed")
				return
			}
		})
		var decodedNilNamespace unstructured.Unstructured
		err = yaml.Unmarshal([]byte(podsGetNilNamespace.Content[0].(map[string]interface{})["text"].(string)), &decodedNilNamespace)
		t.Run("pods_get with name and nil namespace has yaml content", func(t *testing.T) {
			if err != nil {
				t.Fatalf("invalid tool result content %v", err)
				return
			}
		})
		t.Run("pods_get with name and nil namespace returns pod in default", func(t *testing.T) {
			if decodedNilNamespace.GetName() != "a-pod-in-default" {
				t.Fatalf("invalid pod name, expected a-pod-in-default, got %v", decodedNilNamespace.GetName())
				return
			}
			if decodedNilNamespace.GetNamespace() != "default" {
				t.Fatalf("invalid pod namespace, expected default, got %v", decodedNilNamespace.GetNamespace())
				return
			}
		})
		t.Run("pods_get with name and nil namespace omits managed fields", func(t *testing.T) {
			if decodedNilNamespace.GetManagedFields() != nil {
				t.Fatalf("managed fields should be omitted, got %v", decodedNilNamespace.GetManagedFields())
				return
			}
		})
		podsGetInNamespace, err := c.callTool("pods_get", map[string]interface{}{
			"namespace": "ns-1",
			"name":      "a-pod-in-ns-1",
		})
		t.Run("pods_get with name and namespace returns pod", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
				return
			}
			if podsGetInNamespace.IsError {
				t.Fatalf("call tool failed")
				return
			}
		})
		var decodedInNamespace unstructured.Unstructured
		err = yaml.Unmarshal([]byte(podsGetInNamespace.Content[0].(map[string]interface{})["text"].(string)), &decodedInNamespace)
		t.Run("pods_get with name and namespace has yaml content", func(t *testing.T) {
			if err != nil {
				t.Fatalf("invalid tool result content %v", err)
				return
			}
		})
		t.Run("pods_get with name and namespace returns pod in ns-1", func(t *testing.T) {
			if decodedInNamespace.GetName() != "a-pod-in-ns-1" {
				t.Fatalf("invalid pod name, expected a-pod-in-ns-1, got %v", decodedInNamespace.GetName())
				return
			}
			if decodedInNamespace.GetNamespace() != "ns-1" {
				t.Fatalf("invalid pod namespace, ns-1 ns-1, got %v", decodedInNamespace.GetNamespace())
				return
			}
		})
	})
}

func TestPodsDelete(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		// Errors
		t.Run("pods_delete with nil name returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_delete", map[string]interface{}{})
			if toolResult.IsError != true {
				t.Errorf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to delete pod, missing argument name" {
				t.Errorf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		t.Run("pods_delete with not found name returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_delete", map[string]interface{}{"name": "not-found"})
			if toolResult.IsError != true {
				t.Errorf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to delete pod not-found in namespace : pods \"not-found\" not found" {
				t.Errorf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		// Default/nil Namespace
		kc := c.newKubernetesClient()
		_, _ = kc.CoreV1().Pods("default").Create(c.ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "a-pod-to-delete"},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
		}, metav1.CreateOptions{})
		podsDeleteNilNamespace, err := c.callTool("pods_delete", map[string]interface{}{
			"name": "a-pod-to-delete",
		})
		t.Run("pods_delete with name and nil namespace returns success", func(t *testing.T) {
			if err != nil {
				t.Errorf("call tool failed %v", err)
				return
			}
			if podsDeleteNilNamespace.IsError {
				t.Errorf("call tool failed")
				return
			}
			if podsDeleteNilNamespace.Content[0].(map[string]interface{})["text"].(string) != "Pod deleted successfully" {
				t.Errorf("invalid tool result content, got %v", podsDeleteNilNamespace.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		t.Run("pods_delete with name and nil namespace deletes Pod", func(t *testing.T) {
			p, pErr := kc.CoreV1().Pods("default").Get(c.ctx, "a-pod-to-delete", metav1.GetOptions{})
			if pErr == nil && p != nil && p.ObjectMeta.DeletionTimestamp == nil {
				t.Errorf("Pod not deleted")
				return
			}
		})
		// Provided Namespace
		_, _ = kc.CoreV1().Pods("ns-1").Create(c.ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "a-pod-to-delete-in-ns-1"},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
		}, metav1.CreateOptions{})
		podsDeleteInNamespace, err := c.callTool("pods_delete", map[string]interface{}{
			"namespace": "ns-1",
			"name":      "a-pod-to-delete-in-ns-1",
		})
		t.Run("pods_delete with name and namespace returns success", func(t *testing.T) {
			if err != nil {
				t.Errorf("call tool failed %v", err)
				return
			}
			if podsDeleteInNamespace.IsError {
				t.Errorf("call tool failed")
				return
			}
			if podsDeleteInNamespace.Content[0].(map[string]interface{})["text"].(string) != "Pod deleted successfully" {
				t.Errorf("invalid tool result content, got %v", podsDeleteInNamespace.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		t.Run("pods_delete with name and namespace deletes Pod", func(t *testing.T) {
			p, pErr := kc.CoreV1().Pods("ns-1").Get(c.ctx, "a-pod-to-delete-in-ns-1", metav1.GetOptions{})
			if pErr == nil && p != nil && p.ObjectMeta.DeletionTimestamp == nil {
				t.Errorf("Pod not deleted")
				return
			}
		})
		// Managed Pod
		managedLabels := map[string]string{
			"app.kubernetes.io/managed-by": "kubernetes-mcp-server",
			"app.kubernetes.io/name":       "a-manged-pod-to-delete",
		}
		_, _ = kc.CoreV1().Pods("default").Create(c.ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "a-managed-pod-to-delete", Labels: managedLabels},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
		}, metav1.CreateOptions{})
		_, _ = kc.CoreV1().Services("default").Create(c.ctx, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "a-managed-service-to-delete", Labels: managedLabels},
			Spec:       corev1.ServiceSpec{Selector: managedLabels, Ports: []corev1.ServicePort{{Port: 80}}},
		}, metav1.CreateOptions{})
		podsDeleteManaged, err := c.callTool("pods_delete", map[string]interface{}{
			"name": "a-managed-pod-to-delete",
		})
		t.Run("pods_delete with managed pod returns success", func(t *testing.T) {
			if err != nil {
				t.Errorf("call tool failed %v", err)
				return
			}
			if podsDeleteManaged.IsError {
				t.Errorf("call tool failed")
				return
			}
			if podsDeleteManaged.Content[0].(map[string]interface{})["text"].(string) != "Pod deleted successfully" {
				t.Errorf("invalid tool result content, got %v", podsDeleteManaged.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		t.Run("pods_delete with managed pod deletes Pod and Service", func(t *testing.T) {
			p, pErr := kc.CoreV1().Pods("default").Get(c.ctx, "a-managed-pod-to-delete", metav1.GetOptions{})
			if pErr == nil && p != nil && p.ObjectMeta.DeletionTimestamp == nil {
				t.Errorf("Pod not deleted")
				return
			}
			s, sErr := kc.CoreV1().Services("default").Get(c.ctx, "a-managed-service-to-delete", metav1.GetOptions{})
			if sErr == nil && s != nil && s.ObjectMeta.DeletionTimestamp == nil {
				t.Errorf("Service not deleted")
				return
			}
		})
	})
}

func TestPodsDeleteInOpenShift(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		// Managed Pod in OpenShift
		defer c.inOpenShift()() // n.b. two sets of parentheses to invoke the first function
		managedLabels := map[string]string{
			"app.kubernetes.io/managed-by": "kubernetes-mcp-server",
			"app.kubernetes.io/name":       "a-manged-pod-to-delete",
		}
		kc := c.newKubernetesClient()
		_, _ = kc.CoreV1().Pods("default").Create(c.ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "a-managed-pod-to-delete-in-openshift", Labels: managedLabels},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "nginx", Image: "nginx"}}},
		}, metav1.CreateOptions{})
		dynamicClient := dynamic.NewForConfigOrDie(envTestRestConfig)
		_, _ = dynamicClient.Resource(schema.GroupVersionResource{Group: "route.openshift.io", Version: "v1", Resource: "routes"}).
			Namespace("default").Create(c.ctx, &unstructured.Unstructured{Object: map[string]interface{}{
			"apiVersion": "route.openshift.io/v1",
			"kind":       "Route",
			"metadata": map[string]interface{}{
				"name":   "a-managed-route-to-delete",
				"labels": managedLabels,
			},
		}}, metav1.CreateOptions{})
		podsDeleteManagedOpenShift, err := c.callTool("pods_delete", map[string]interface{}{
			"name": "a-managed-pod-to-delete-in-openshift",
		})
		t.Run("pods_delete with managed pod in OpenShift returns success", func(t *testing.T) {
			if err != nil {
				t.Errorf("call tool failed %v", err)
				return
			}
			if podsDeleteManagedOpenShift.IsError {
				t.Errorf("call tool failed")
				return
			}
			if podsDeleteManagedOpenShift.Content[0].(map[string]interface{})["text"].(string) != "Pod deleted successfully" {
				t.Errorf("invalid tool result content, got %v", podsDeleteManagedOpenShift.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		t.Run("pods_delete with managed pod in OpenShift deletes Pod and Route", func(t *testing.T) {
			p, pErr := kc.CoreV1().Pods("default").Get(c.ctx, "a-managed-pod-to-delete-in-openshift", metav1.GetOptions{})
			if pErr == nil && p != nil && p.ObjectMeta.DeletionTimestamp == nil {
				t.Errorf("Pod not deleted")
				return
			}
			r, rErr := dynamicClient.
				Resource(schema.GroupVersionResource{Group: "route.openshift.io", Version: "v1", Resource: "routes"}).
				Namespace("default").Get(c.ctx, "a-managed-route-to-delete", metav1.GetOptions{})
			if rErr == nil && r != nil && r.GetDeletionTimestamp() == nil {
				t.Errorf("Route not deleted")
				return
			}
		})
	})
}

func TestPodsLog(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		t.Run("pods_log with nil name returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_log", map[string]interface{}{})
			if toolResult.IsError != true {
				t.Fatalf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to get pod log, missing argument name" {
				t.Fatalf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		t.Run("pods_log with not found name returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_log", map[string]interface{}{"name": "not-found"})
			if toolResult.IsError != true {
				t.Fatalf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to get pod not-found log in namespace : pods \"not-found\" not found" {
				t.Fatalf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		podsLogNilNamespace, err := c.callTool("pods_log", map[string]interface{}{
			"name": "a-pod-in-default",
		})
		t.Run("pods_log with name and nil namespace returns pod log", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
				return
			}
			if podsLogNilNamespace.IsError {
				t.Fatalf("call tool failed")
				return
			}
		})
		podsLogInNamespace, err := c.callTool("pods_log", map[string]interface{}{
			"namespace": "ns-1",
			"name":      "a-pod-in-ns-1",
		})
		t.Run("pods_log with name and namespace returns pod log", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
				return
			}
			if podsLogInNamespace.IsError {
				t.Fatalf("call tool failed")
				return
			}
		})
	})
}

func TestPodsRun(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		t.Run("pods_run with nil image returns error", func(t *testing.T) {
			toolResult, _ := c.callTool("pods_run", map[string]interface{}{})
			if toolResult.IsError != true {
				t.Errorf("call tool should fail")
				return
			}
			if toolResult.Content[0].(map[string]interface{})["text"].(string) != "failed to run pod, missing argument image" {
				t.Errorf("invalid error message, got %v", toolResult.Content[0].(map[string]interface{})["text"].(string))
				return
			}
		})
		podsRunNilNamespace, err := c.callTool("pods_run", map[string]interface{}{"image": "nginx"})
		t.Run("pods_run with image and nil namespace runs pod", func(t *testing.T) {
			if err != nil {
				t.Errorf("call tool failed %v", err)
				return
			}
			if podsRunNilNamespace.IsError {
				t.Errorf("call tool failed")
				return
			}
		})
		var decodedNilNamespace []unstructured.Unstructured
		err = yaml.Unmarshal([]byte(podsRunNilNamespace.Content[0].(map[string]interface{})["text"].(string)), &decodedNilNamespace)
		t.Run("pods_run with image and nil namespace has yaml content", func(t *testing.T) {
			if err != nil {
				t.Errorf("invalid tool result content %v", err)
				return
			}
		})
		t.Run("pods_run with image and nil namespace returns 1 item (Pod)", func(t *testing.T) {
			if len(decodedNilNamespace) != 1 {
				t.Errorf("invalid pods count, expected 1, got %v", len(decodedNilNamespace))
				return
			}
			if decodedNilNamespace[0].GetKind() != "Pod" {
				t.Errorf("invalid pod kind, expected Pod, got %v", decodedNilNamespace[0].GetKind())
				return
			}
		})
		t.Run("pods_run with image and nil namespace returns pod in default", func(t *testing.T) {
			if decodedNilNamespace[0].GetNamespace() != "default" {
				t.Errorf("invalid pod namespace, expected default, got %v", decodedNilNamespace[0].GetNamespace())
				return
			}
		})
		t.Run("pods_run with image and nil namespace returns pod with random name", func(t *testing.T) {
			if !strings.HasPrefix(decodedNilNamespace[0].GetName(), "kubernetes-mcp-server-run-") {
				t.Errorf("invalid pod name, expected random, got %v", decodedNilNamespace[0].GetName())
				return
			}
		})
		t.Run("pods_run with image and nil namespace returns pod with labels", func(t *testing.T) {
			labels := decodedNilNamespace[0].Object["metadata"].(map[string]interface{})["labels"].(map[string]interface{})
			if labels["app.kubernetes.io/name"] == "" {
				t.Errorf("invalid labels, expected app.kubernetes.io/name, got %v", labels)
				return
			}
			if labels["app.kubernetes.io/component"] == "" {
				t.Errorf("invalid labels, expected app.kubernetes.io/component, got %v", labels)
				return
			}
			if labels["app.kubernetes.io/managed-by"] != "kubernetes-mcp-server" {
				t.Errorf("invalid labels, expected app.kubernetes.io/managed-by, got %v", labels)
				return
			}
			if labels["app.kubernetes.io/part-of"] != "kubernetes-mcp-server-run-sandbox" {
				t.Errorf("invalid labels, expected app.kubernetes.io/part-of, got %v", labels)
				return
			}
		})
		t.Run("pods_run with image and nil namespace returns pod with nginx container", func(t *testing.T) {
			containers := decodedNilNamespace[0].Object["spec"].(map[string]interface{})["containers"].([]interface{})
			if containers[0].(map[string]interface{})["image"] != "nginx" {
				t.Errorf("invalid container name, expected nginx, got %v", containers[0].(map[string]interface{})["image"])
				return
			}
		})

		podsRunNamespaceAndPort, err := c.callTool("pods_run", map[string]interface{}{"image": "nginx", "port": 80})
		t.Run("pods_run with image, namespace, and port runs pod", func(t *testing.T) {
			if err != nil {
				t.Errorf("call tool failed %v", err)
				return
			}
			if podsRunNamespaceAndPort.IsError {
				t.Errorf("call tool failed")
				return
			}
		})
		var decodedNamespaceAndPort []unstructured.Unstructured
		err = yaml.Unmarshal([]byte(podsRunNamespaceAndPort.Content[0].(map[string]interface{})["text"].(string)), &decodedNamespaceAndPort)
		t.Run("pods_run with image, namespace, and port has yaml content", func(t *testing.T) {
			if err != nil {
				t.Errorf("invalid tool result content %v", err)
				return
			}
		})
		t.Run("pods_run with image, namespace, and port returns 2 items (Pod + Service)", func(t *testing.T) {
			if len(decodedNamespaceAndPort) != 2 {
				t.Errorf("invalid pods count, expected 2, got %v", len(decodedNamespaceAndPort))
				return
			}
			if decodedNamespaceAndPort[0].GetKind() != "Pod" {
				t.Errorf("invalid pod kind, expected Pod, got %v", decodedNamespaceAndPort[0].GetKind())
				return
			}
			if decodedNamespaceAndPort[1].GetKind() != "Service" {
				t.Errorf("invalid service kind, expected Service, got %v", decodedNamespaceAndPort[1].GetKind())
				return
			}
		})
		t.Run("pods_run with image, namespace, and port returns pod with port", func(t *testing.T) {
			containers := decodedNamespaceAndPort[0].Object["spec"].(map[string]interface{})["containers"].([]interface{})
			ports := containers[0].(map[string]interface{})["ports"].([]interface{})
			if ports[0].(map[string]interface{})["containerPort"] != int64(80) {
				t.Errorf("invalid container port, expected 80, got %v", ports[0].(map[string]interface{})["containerPort"])
				return
			}
		})
		t.Run("pods_run with image, namespace, and port returns service with port and selector", func(t *testing.T) {
			ports := decodedNamespaceAndPort[1].Object["spec"].(map[string]interface{})["ports"].([]interface{})
			if ports[0].(map[string]interface{})["port"] != int64(80) {
				t.Errorf("invalid service port, expected 80, got %v", ports[0].(map[string]interface{})["port"])
				return
			}
			if ports[0].(map[string]interface{})["targetPort"] != int64(80) {
				t.Errorf("invalid service target port, expected 80, got %v", ports[0].(map[string]interface{})["targetPort"])
				return
			}
			selector := decodedNamespaceAndPort[1].Object["spec"].(map[string]interface{})["selector"].(map[string]interface{})
			if selector["app.kubernetes.io/name"] == "" {
				t.Errorf("invalid service selector, expected app.kubernetes.io/name, got %v", selector)
				return
			}
			if selector["app.kubernetes.io/managed-by"] != "kubernetes-mcp-server" {
				t.Errorf("invalid service selector, expected app.kubernetes.io/managed-by, got %v", selector)
				return
			}
			if selector["app.kubernetes.io/part-of"] != "kubernetes-mcp-server-run-sandbox" {
				t.Errorf("invalid service selector, expected app.kubernetes.io/part-of, got %v", selector)
				return
			}
		})
	})
}

func TestPodsRunInOpenShift(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		defer c.inOpenShift()() // n.b. two sets of parentheses to invoke the first function
		t.Run("pods_run with image, namespace, and port returns route with port", func(t *testing.T) {
			podsRunInOpenShift, err := c.callTool("pods_run", map[string]interface{}{"image": "nginx", "port": 80})
			if err != nil {
				t.Errorf("call tool failed %v", err)
				return
			}
			if podsRunInOpenShift.IsError {
				t.Errorf("call tool failed")
				return
			}
			var decodedPodServiceRoute []unstructured.Unstructured
			err = yaml.Unmarshal([]byte(podsRunInOpenShift.Content[0].(map[string]interface{})["text"].(string)), &decodedPodServiceRoute)
			if err != nil {
				t.Errorf("invalid tool result content %v", err)
				return
			}
			if len(decodedPodServiceRoute) != 3 {
				t.Errorf("invalid pods count, expected 3, got %v", len(decodedPodServiceRoute))
				return
			}
			if decodedPodServiceRoute[2].GetKind() != "Route" {
				t.Errorf("invalid route kind, expected Route, got %v", decodedPodServiceRoute[2].GetKind())
				return
			}
			targetPort := decodedPodServiceRoute[2].Object["spec"].(map[string]interface{})["port"].(map[string]interface{})["targetPort"].(int64)
			if targetPort != 80 {
				t.Errorf("invalid route target port, expected 80, got %v", targetPort)
				return
			}
		})
	})
}
