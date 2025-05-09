package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
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

func TestTools(t *testing.T) {
	expectedNames := []string{
		"configuration_view",
		"events_list",
		"helm_list",
		"namespaces_list",
		"pods_list",
		"pods_list_in_namespace",
		"pods_get",
		"pods_delete",
		"pods_log",
		"pods_run",
		"pods_exec",
		"resources_list",
		"resources_get",
		"resources_create_or_update",
		"resources_delete",
	}
	testCase(t, func(c *mcpContext) {
		tools, err := c.mcpClient.ListTools(c.ctx, mcp.ListToolsRequest{})
		t.Run("ListTools returns tools", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call ListTools failed %v", err)
				return
			}
		})
		nameSet := make(map[string]bool)
		for _, tool := range tools.Tools {
			nameSet[tool.Name] = true
		}
		for _, name := range expectedNames {
			t.Run("ListTools has "+name+" tool", func(t *testing.T) {
				if nameSet[name] != true {
					t.Fatalf("tool %s not found", name)
					return
				}
			})
		}
	})
}

func TestToolsInOpenShift(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		defer c.inOpenShift()() // n.b. two sets of parentheses to invoke the first function
		c.mcpServer.server.AddTools(c.mcpServer.initNamespaces()...)
		c.mcpServer.server.AddTools(c.mcpServer.initResources()...)
		tools, err := c.mcpClient.ListTools(c.ctx, mcp.ListToolsRequest{})
		t.Run("ListTools returns tools", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call ListTools failed %v", err)
			}
		})
		t.Run("ListTools contains projects_list tool", func(t *testing.T) {
			idx := slices.IndexFunc(tools.Tools, func(tool mcp.Tool) bool {
				return tool.Name == "projects_list"
			})
			if idx == -1 {
				t.Fatalf("tool projects_list not found")
			}
		})
		t.Run("ListTools has resources_list tool with OpenShift hint", func(t *testing.T) {
			idx := slices.IndexFunc(tools.Tools, func(tool mcp.Tool) bool {
				return tool.Name == "resources_list"
			})
			if idx == -1 {
				t.Fatalf("tool resources_list not found")
			}
			if !strings.Contains(tools.Tools[idx].Description, ", route.openshift.io/v1 Route") {
				t.Fatalf("tool resources_list does not have OpenShift hint, got %s", tools.Tools[9].Description)
			}
		})
	})

}
