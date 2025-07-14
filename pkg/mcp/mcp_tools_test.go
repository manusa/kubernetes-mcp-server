package mcp

import (
	"k8s.io/utils/ptr"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
)

func TestUnrestricted(t *testing.T) {
	testCase(t, func(c *mcpContext) {
		tools, err := c.mcpClient.ListTools(c.ctx, mcp.ListToolsRequest{})
		t.Run("ListTools returns tools", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call ListTools failed %v", err)
			}
		})
		t.Run("Destructive tools ARE NOT read only", func(t *testing.T) {
			for _, tool := range tools.Tools {
				readOnly := ptr.Deref(tool.Annotations.ReadOnlyHint, false)
				destructive := ptr.Deref(tool.Annotations.DestructiveHint, false)
				if readOnly && destructive {
					t.Errorf("Tool %s is read-only and destructive, which is not allowed", tool.Name)
				}
			}
		})
	})
}

func TestReadOnly(t *testing.T) {
	readOnlyServer := func(c *mcpContext) { c.staticConfig = &config.StaticConfig{ReadOnly: true} }
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
	disableDestructiveServer := func(c *mcpContext) { c.staticConfig = &config.StaticConfig{DisableDestructive: true} }
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

func TestEnabledTools(t *testing.T) {
	testCaseWithContext(t, &mcpContext{
		staticConfig: &config.StaticConfig{
			EnabledTools: []string{"namespaces_list", "events_list"},
		},
	}, func(c *mcpContext) {
		tools, err := c.mcpClient.ListTools(c.ctx, mcp.ListToolsRequest{})
		t.Run("ListTools returns tools", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call ListTools failed %v", err)
			}
		})
		t.Run("ListTools returns only explicitly enabled tools", func(t *testing.T) {
			if len(tools.Tools) != 2 {
				t.Fatalf("ListTools should return 2 tools, got %d", len(tools.Tools))
			}
			for _, tool := range tools.Tools {
				if tool.Name != "namespaces_list" && tool.Name != "events_list" {
					t.Errorf("Tool %s is not enabled but should be", tool.Name)
				}
			}
		})
	})
}

func TestDisabledTools(t *testing.T) {
	testCaseWithContext(t, &mcpContext{
		staticConfig: &config.StaticConfig{
			DisabledTools: []string{"namespaces_list", "events_list"},
		},
	}, func(c *mcpContext) {
		tools, err := c.mcpClient.ListTools(c.ctx, mcp.ListToolsRequest{})
		t.Run("ListTools returns tools", func(t *testing.T) {
			if err != nil {
				t.Fatalf("call ListTools failed %v", err)
			}
		})
		t.Run("ListTools does not return disabled tools", func(t *testing.T) {
			for _, tool := range tools.Tools {
				if tool.Name == "namespaces_list" || tool.Name == "events_list" {
					t.Errorf("Tool %s is not disabled but should be", tool.Name)
				}
			}
		})
	})
}
