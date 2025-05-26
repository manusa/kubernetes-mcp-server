package mcp

import (
	"context"
	"github.com/mark3labs/mcp-go/mcp"
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
