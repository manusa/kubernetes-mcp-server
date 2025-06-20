package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadConfigMissingFile(t *testing.T) {
	config, err := ReadConfig("non-existent-config.toml")
	t.Run("returns error for missing file", func(t *testing.T) {
		if err == nil {
			t.Fatal("Expected error for missing file, got nil")
		}
		if config != nil {
			t.Fatalf("Expected nil config for missing file, got %v", config)
		}
	})
}

func TestReadConfigInvalid(t *testing.T) {
	invalidConfigPath := writeConfig(t, `
[[denied_resources]]
group = "apps"
version = "v1"
kind = "Deployment"
[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
kind = "Role
`)

	config, err := ReadConfig(invalidConfigPath)
	t.Run("returns error for invalid file", func(t *testing.T) {
		if err == nil {
			t.Fatal("Expected error for invalid file, got nil")
		}
		if config != nil {
			t.Fatalf("Expected nil config for invalid file, got %v", config)
		}
	})
	t.Run("error message contains toml error with line number", func(t *testing.T) {
		expectedError := "toml: line 9"
		if err != nil && !strings.HasPrefix(err.Error(), expectedError) {
			t.Fatalf("Expected error message '%s' to contain line number, got %v", expectedError, err)
		}
	})
}

func TestReadConfigValid(t *testing.T) {
	validConfigPath := writeConfig(t, `
[[denied_resources]]
group = "apps"
version = "v1"
kind = "Deployment"

[[denied_resources]]
group = "rbac.authorization.k8s.io"
version = "v1"
`)

	config, err := ReadConfig(validConfigPath)
	t.Run("reads and unmarshalls file", func(t *testing.T) {
		if err != nil {
			t.Fatalf("ReadConfig returned an error for a valid file: %v", err)
		}
		if config == nil {
			t.Fatal("ReadConfig returned a nil config for a valid file")
		}
	})
	t.Run("denied resources are parsed correctly", func(t *testing.T) {
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

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "config.toml")
	err := os.WriteFile(path, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write config file %s: %v", path, err)
	}
	return path
}
