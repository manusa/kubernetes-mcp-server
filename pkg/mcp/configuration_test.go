package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
	"strings"
	"testing"
)

func TestConfigurationView(t *testing.T) {
	t.Run("configuration_view returns configuration", testCase(func(t *testing.T, c *mcpContext) {
		configurationGet := mcp.CallToolRequest{}
		configurationGet.Params.Name = "configuration_view"
		configurationGet.Params.Arguments = map[string]interface{}{}
		tools, err := c.mcpClient.CallTool(c.ctx, configurationGet)
		if err != nil {
			t.Fatalf("call tool failed %v", err)
			return
		}
		resultContent := tools.Content[0].(map[string]interface{})["text"].(string)
		if !strings.Contains(resultContent, "cluster: fake\n") {
			t.Fatalf("mismatch in kube config: %s", resultContent)
			return
		}
	}))
}
