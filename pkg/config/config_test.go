package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("ValidConfigFileWithDeniedResources", func(t *testing.T) {
		validConfigContent := `
[[denied_resources]]
group = "apps"
version = "v1"
kind = "Deployment"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
`
		validConfigPath := filepath.Join(tempDir, "valid_denied_config.toml")
		err := os.WriteFile(validConfigPath, []byte(validConfigContent), 0644)
		if err != nil {
			t.Fatalf("Failed to write valid config file: %v", err)
		}

		config, err := ReadConfig(validConfigPath)
		if err != nil {
			t.Fatalf("ReadConfig returned an error for a valid file: %v", err)
		}

		if config == nil {
			t.Fatal("ReadConfig returned a nil config for a valid file")
		}

		if len(config.DeniedResources) != 2 {
			t.Fatalf("Expected 2 denied resources, got %d", len(config.DeniedResources))
		}

		if config.DeniedResources[0].Group != "apps" ||
			config.DeniedResources[0].Version != "v1" ||
			config.DeniedResources[0].Kind != "Deployment" {
			t.Errorf("Unexpected denied resources: %v", config.DeniedResources[0])
		}
	})
}
