package mcp

import (
	"encoding/base64"
	"github.com/mark3labs/mcp-go/mcp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
	"testing"
)

func TestHelmList(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		c.withEnvTest()
		kc := c.newKubernetesClient()
		_ = kc.CoreV1().Secrets("default").Delete(c.ctx, "release-to-list", metav1.DeleteOptions{})
		toolResult, err := c.callTool("helm_list", map[string]interface{}{})
		t.Run("helm_list with no releases, returns not found", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
			}
			if toolResult.IsError {
				t.Fatalf("call tool failed")
			}
			if toolResult.Content[0].(mcp.TextContent).Text != "No Helm releases found" {
				t.Fatalf("unexpected result %v", toolResult.Content[0].(mcp.TextContent).Text)
			}
		})
		_, err = kc.CoreV1().Secrets("default").Create(c.ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:   "release-to-list",
				Labels: map[string]string{"owner": "helm"},
			},
			Data: map[string][]byte{
				"release": []byte(base64.StdEncoding.EncodeToString([]byte("{" +
					"\"name\":\"release-to-list\"," +
					"\"info\":{\"status\":\"deployed\"}" +
					"}"))),
			},
		}, metav1.CreateOptions{})
		toolResult, err = c.callTool("helm_list", map[string]interface{}{})
		t.Run("helm_list with deployed release, returns release", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
			}
			if toolResult.IsError {
				t.Fatalf("call tool failed")
			}
			var decoded []map[string]interface{}
			err = yaml.Unmarshal([]byte(toolResult.Content[0].(mcp.TextContent).Text), &decoded)
			if err != nil {
				t.Fatalf("invalid tool result content %v", err)
			}
			if len(decoded) != 1 {
				t.Fatalf("invalid helm list count, expected 1, got %v", len(decoded))
			}
			if decoded[0]["name"] != "release-to-list" {
				t.Fatalf("invalid helm list name, expected release-to-list, got %v", decoded[0]["name"])
			}
			if decoded[0]["info"].(map[string]interface{})["status"] != "deployed" {
				t.Fatalf("invalid helm list status, expected deployed, got %v", decoded[0]["info"].(map[string]interface{})["status"])
			}
		})
		toolResult, err = c.callTool("helm_list", map[string]interface{}{"namespace": "ns-1"})
		t.Run("helm_list with deployed release in other namespaces, returns not found", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
			}
			if toolResult.IsError {
				t.Fatalf("call tool failed")
			}
			if toolResult.Content[0].(mcp.TextContent).Text != "No Helm releases found" {
				t.Fatalf("unexpected result %v", toolResult.Content[0].(mcp.TextContent).Text)
			}
		})
		toolResult, err = c.callTool("helm_list", map[string]interface{}{"namespace": "ns-1", "all_namespaces": true})
		t.Run("helm_list with deployed release in all namespaces, returns release", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call tool failed %v", err)
			}
			if toolResult.IsError {
				t.Fatalf("call tool failed")
			}
			var decoded []map[string]interface{}
			err = yaml.Unmarshal([]byte(toolResult.Content[0].(mcp.TextContent).Text), &decoded)
			if err != nil {
				t.Fatalf("invalid tool result content %v", err)
			}
			if len(decoded) != 1 {
				t.Fatalf("invalid helm list count, expected 1, got %v", len(decoded))
			}
			if decoded[0]["name"] != "release-to-list" {
				t.Fatalf("invalid helm list name, expected release-to-list, got %v", decoded[0]["name"])
			}
			if decoded[0]["info"].(map[string]interface{})["status"] != "deployed" {
				t.Fatalf("invalid helm list status, expected deployed, got %v", decoded[0]["info"].(map[string]interface{})["status"])
			}
		})
	})
}
