package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestWatchKubeConfig(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("Skipping test on non-linux platforms")
	}
	testCase(t, func(c *mcpContext) {
		// Given
		withTimeout, cancel := context.WithTimeout(c.ctx, 5*time.Second)
		defer cancel()
		var notification *mcp.JSONRPCNotification
		c.mcpClient.OnNotification(func(n mcp.JSONRPCNotification) {
			notification = &n
		})
		// When
		f, _ := os.OpenFile(filepath.Join(c.tempDir, "config"), os.O_APPEND|os.O_WRONLY, 0644)
		_, _ = f.WriteString("\n")
		for {
			if notification != nil {
				break
			}
			select {
			case <-withTimeout.Done():
				break
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}
		// Then
		t.Run("WatchKubeConfig notifies tools change", func(t *testing.T) {
			if notification == nil {
				t.Fatalf("WatchKubeConfig did not notify")
			}
			if notification.Method != "notifications/tools/list_changed" {
				t.Fatalf("WatchKubeConfig did not notify tools change, got %s", notification.Method)
			}
		})
	})
}

func TestReadOnly(t *testing.T) {
	readOnlyServer := func(c *mcpContext) { c.readOnly = true }
	testCaseWithContext(t, &mcpContext{before: readOnlyServer}, func(c *mcpContext) {
		tools, err := c.mcpClient.ListTools(c.ctx, mcp.ListToolsRequest{})
		t.Run("ListTools returns tools", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call ListTools failed %v", err)
			}
		})
		t.Run("ListTools returns only read-only tools", func(t *testing.T) {
			for _, tool := range tools.Tools {
				if tool.Annotations.ReadOnlyHint == nil || !*tool.Annotations.ReadOnlyHint {
					t.Errorf("Tool %s is not read-only but should be", tool.Name)
				}
				if tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint {
					t.Errorf("Tool %s is destructive but should not be in read-only mode", tool.Name)
				}
			}
		})
	})
}

func TestDisableDestructive(t *testing.T) {
	disableDestructiveServer := func(c *mcpContext) { c.disableDestructive = true }
	testCaseWithContext(t, &mcpContext{before: disableDestructiveServer}, func(c *mcpContext) {
		tools, err := c.mcpClient.ListTools(c.ctx, mcp.ListToolsRequest{})
		t.Run("ListTools returns tools", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call ListTools failed %v", err)
			}
		})
		t.Run("ListTools does not return destructive tools", func(t *testing.T) {
			for _, tool := range tools.Tools {
				if tool.Annotations.DestructiveHint != nil && *tool.Annotations.DestructiveHint {
					t.Errorf("Tool %s is destructive but should not be", tool.Name)
				}
			}
		})
	})
}

func TestSseHeaders(t *testing.T) {
	mockServer := NewMockServer()
	defer mockServer.Close()
	before := func(c *mcpContext) {
		c.withKubeConfig(mockServer.config)
		c.clientOptions = append(c.clientOptions, client.WithHeaders(map[string]string{"kubernetes-authorization": "Bearer a-token-from-mcp-client"}))
	}
	pathHeaders := make(map[string]http.Header, 0)
	mockServer.Handle(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		pathHeaders[req.URL.Path] = req.Header.Clone()
		// Request Performed by DiscoveryClient to Kube API (Get API Groups legacy -core-)
		if req.URL.Path == "/api" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"APIVersions","versions":["v1"],"serverAddressByClientCIDRs":[{"clientCIDR":"0.0.0.0/0"}]}`))
			return
		}
		// Request Performed by DiscoveryClient to Kube API (Get API Groups)
		if req.URL.Path == "/apis" {
			w.Header().Set("Content-Type", "application/json")
			//w.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[{"name":"apps","versions":[{"groupVersion":"apps/v1","version":"v1"}],"preferredVersion":{"groupVersion":"apps/v1","version":"v1"}}]}`))
			_, _ = w.Write([]byte(`{"kind":"APIGroupList","apiVersion":"v1","groups":[]}`))
			return
		}
		// Request Performed by DiscoveryClient to Kube API (Get API Resources)
		if req.URL.Path == "/api/v1" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"APIResourceList","apiVersion":"v1","resources":[{"name":"pods","singularName":"","namespaced":true,"kind":"Pod","verbs":["get","list","watch","create","update","patch","delete"]}]}`))
			return
		}
		// Request Performed by DynamicClient
		if req.URL.Path == "/api/v1/namespaces/default/pods" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"kind":"PodList","apiVersion":"v1","items":[]}`))
			return
		}
		// Request Performed by kubernetes.Interface
		if req.URL.Path == "/api/v1/namespaces/default/pods/a-pod-to-delete" {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(404)
	}))
	testCaseWithContext(t, &mcpContext{before: before}, func(c *mcpContext) {
		c.callTool("pods_list", map[string]interface{}{})
		t.Run("DiscoveryClient propagates headers to Kube API", func(t *testing.T) {
			if len(pathHeaders) == 0 {
				t.Fatalf("No requests were made to Kube API")
			}
			if pathHeaders["/api"] == nil || pathHeaders["/api"].Get("Authorization") != "Bearer a-token-from-mcp-client" {
				t.Fatalf("Overridden header Authorization not found in request to /api")
			}
			if pathHeaders["/apis"] == nil || pathHeaders["/apis"].Get("Authorization") != "Bearer a-token-from-mcp-client" {
				t.Fatalf("Overridden header Authorization not found in request to /apis")
			}
			if pathHeaders["/api/v1"] == nil || pathHeaders["/api/v1"].Get("Authorization") != "Bearer a-token-from-mcp-client" {
				t.Fatalf("Overridden header Authorization not found in request to /api/v1")
			}
		})
		t.Run("DynamicClient propagates headers to Kube API", func(t *testing.T) {
			if len(pathHeaders) == 0 {
				t.Fatalf("No requests were made to Kube API")
			}
			if pathHeaders["/api/v1/namespaces/default/pods"] == nil || pathHeaders["/api/v1/namespaces/default/pods"].Get("Authorization") != "Bearer a-token-from-mcp-client" {
				t.Fatalf("Overridden header Authorization not found in request to /api/v1/namespaces/default/pods")
			}
		})
		c.callTool("pods_delete", map[string]interface{}{"name": "a-pod-to-delete"})
		t.Run("kubernetes.Interface propagates headers to Kube API", func(t *testing.T) {
			if len(pathHeaders) == 0 {
				t.Fatalf("No requests were made to Kube API")
			}
			if pathHeaders["/api/v1/namespaces/default/pods/a-pod-to-delete"] == nil || pathHeaders["/api/v1/namespaces/default/pods/a-pod-to-delete"].Get("Authorization") != "Bearer a-token-from-mcp-client" {
				t.Fatalf("Overridden header Authorization not found in request to /api/v1/namespaces/default/pods/a-pod-to-delete")
			}
		})
	})
}
