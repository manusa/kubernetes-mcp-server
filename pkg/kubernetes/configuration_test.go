package kubernetes

import (
	"errors"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd/api"
	v1 "k8s.io/client-go/tools/clientcmd/api/v1"
	"sigs.k8s.io/yaml"
)

func TestKubernetes_IsInCluster(t *testing.T) {
	t.Run("with explicit kubeconfig", func(t *testing.T) {
		k := Kubernetes{
			Kubeconfig: "kubeconfig",
		}
		if k.IsInCluster() {
			t.Errorf("expected not in cluster, got in cluster")
		}
	})
	t.Run("with empty kubeconfig and in cluster", func(t *testing.T) {
		originalFunction := InClusterConfig
		InClusterConfig = func() (*rest.Config, error) {
			return &rest.Config{}, nil
		}
		defer func() {
			InClusterConfig = originalFunction
		}()
		k := Kubernetes{
			Kubeconfig: "",
		}
		if !k.IsInCluster() {
			t.Errorf("expected in cluster, got not in cluster")
		}
	})
	t.Run("with empty kubeconfig and not in cluster (empty)", func(t *testing.T) {
		originalFunction := InClusterConfig
		InClusterConfig = func() (*rest.Config, error) {
			return nil, nil
		}
		defer func() {
			InClusterConfig = originalFunction
		}()
		k := Kubernetes{
			Kubeconfig: "",
		}
		if k.IsInCluster() {
			t.Errorf("expected not in cluster, got in cluster")
		}
	})
	t.Run("with empty kubeconfig and not in cluster (error)", func(t *testing.T) {
		originalFunction := InClusterConfig
		InClusterConfig = func() (*rest.Config, error) {
			return nil, errors.New("error")
		}
		defer func() {
			InClusterConfig = originalFunction
		}()
		k := Kubernetes{
			Kubeconfig: "",
		}
		if k.IsInCluster() {
			t.Errorf("expected not in cluster, got in cluster")
		}
	})
}

func TestKubernetes_ResolveKubernetesConfigurations_Explicit(t *testing.T) {
	t.Run("with missing file", func(t *testing.T) {
		if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
			t.Skip("Skipping test on non-linux platforms")
		}
		tempDir := t.TempDir()
		k := Kubernetes{Kubeconfig: path.Join(tempDir, "config")}
		err := resolveKubernetesConfigurations(&k)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected file not found error, got %v", err)
		}
		if !strings.HasSuffix(err.Error(), ": no such file or directory") {
			t.Errorf("expected file not found error, got %v", err)
		}
	})
	t.Run("with empty file", func(t *testing.T) {
		tempDir := t.TempDir()
		kubeconfigPath := path.Join(tempDir, "config")
		if err := os.WriteFile(kubeconfigPath, []byte(""), 0644); err != nil {
			t.Fatalf("failed to create kubeconfig file: %v", err)
		}
		k := Kubernetes{Kubeconfig: kubeconfigPath}
		err := resolveKubernetesConfigurations(&k)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no configuration has been provided") {
			t.Errorf("expected no kubeconfig error, got %v", err)
		}
	})
	t.Run("with valid file", func(t *testing.T) {
		tempDir := t.TempDir()
		kubeconfigPath := path.Join(tempDir, "config")
		kubeconfigContent := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://example.com
  name: example-cluster
contexts:
- context:
    cluster: example-cluster
    user: example-user
  name: example-context
current-context: example-context
users:
- name: example-user
  user:
    token: example-token
`
		if err := os.WriteFile(kubeconfigPath, []byte(kubeconfigContent), 0644); err != nil {
			t.Fatalf("failed to create kubeconfig file: %v", err)
		}
		k := Kubernetes{Kubeconfig: kubeconfigPath}
		err := resolveKubernetesConfigurations(&k)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if k.cfg == nil {
			t.Errorf("expected non-nil config, got nil")
		}
		if k.cfg.Host != "https://example.com" {
			t.Errorf("expected host https://example.com, got %s", k.cfg.Host)
		}
	})
}

func TestConfigurationViewWithOIDC(t *testing.T) {
	// Save the original InClusterConfig function
	originalInClusterConfig := InClusterConfig

	// Mock InClusterConfig to return a config with OIDC auth provider
	InClusterConfig = func() (*rest.Config, error) {
		return &rest.Config{
			Host: "https://kubernetes.default.svc",
			AuthProvider: &api.AuthProviderConfig{
				Name: "oidc",
				Config: map[string]string{
					"client-id":      "test-client",
					"client-secret":  "test-secret",
					"id-token":       "test-id-token",
					"refresh-token":  "test-refresh-token",
					"idp-issuer-url": "https://example.org",
				},
			},
		}, nil
	}

	// Restore the original function when the test completes
	defer func() {
		InClusterConfig = originalInClusterConfig
	}()

	// Create a Kubernetes instance with empty kubeconfig to force in-cluster mode
	k := &Kubernetes{
		Kubeconfig: "",
	}

	// Initialize cfg directly to prevent nil pointer dereference
	k.cfg, _ = InClusterConfig()

	// Call ConfigurationView
	configYaml, err := k.ConfigurationView(true)
	if err != nil {
		t.Fatalf("ConfigurationView failed: %v", err)
	}

	// Parse the YAML
	var config v1.Config
	if err := yaml.Unmarshal([]byte(configYaml), &config); err != nil {
		t.Fatalf("Failed to parse config YAML: %v", err)
	}

	// Verify the auth provider is included
	if len(config.AuthInfos) != 1 {
		t.Fatalf("Expected 1 auth info, got %d", len(config.AuthInfos))
	}

	authInfo := config.AuthInfos[0]
	if authInfo.Name != "user" {
		t.Errorf("Expected auth info name to be 'user', got '%s'", authInfo.Name)
	}

	if authInfo.AuthInfo.AuthProvider == nil {
		t.Fatalf("Expected auth provider to be present, got nil")
	}

	if authInfo.AuthInfo.AuthProvider.Name != "oidc" {
		t.Errorf("Expected auth provider name to be 'oidc', got '%s'", authInfo.AuthInfo.AuthProvider.Name)
	}

	// Verify the auth provider config
	authProviderConfig := authInfo.AuthInfo.AuthProvider.Config
	expectedKeys := []string{"client-id", "client-secret", "id-token", "refresh-token", "idp-issuer-url"}
	for _, key := range expectedKeys {
		if value, exists := authProviderConfig[key]; !exists {
			t.Errorf("Expected auth provider config to have key '%s', but it was missing", key)
		} else if key == "client-id" && value != "test-client" {
			t.Errorf("Expected auth provider config key '%s' to have value 'test-client', got '%s'", key, value)
		}
	}
}

func TestConfigurationViewWithExecProvider(t *testing.T) {
	// Save the original InClusterConfig function
	originalInClusterConfig := InClusterConfig

	// Mock InClusterConfig to return a config with ExecProvider
	InClusterConfig = func() (*rest.Config, error) {
		return &rest.Config{
			Host: "https://kubernetes.default.svc",
			ExecProvider: &api.ExecConfig{
				Command: "aws",
				Args:    []string{"eks", "get-token", "--cluster-name", "test-cluster"},
				Env: []api.ExecEnvVar{
					{
						Name:  "AWS_PROFILE",
						Value: "test-profile",
					},
				},
				APIVersion: "client.authentication.k8s.io/v1beta1",
			},
		}, nil
	}

	// Restore the original function when the test completes
	defer func() {
		InClusterConfig = originalInClusterConfig
	}()

	// Create a Kubernetes instance with empty kubeconfig to force in-cluster mode
	k := &Kubernetes{
		Kubeconfig: "",
	}

	// Initialize cfg directly to prevent nil pointer dereference
	k.cfg, _ = InClusterConfig()

	// Call ConfigurationView
	configYaml, err := k.ConfigurationView(true)
	if err != nil {
		t.Fatalf("ConfigurationView failed: %v", err)
	}

	// Parse the YAML
	var config v1.Config
	if err := yaml.Unmarshal([]byte(configYaml), &config); err != nil {
		t.Fatalf("Failed to parse config YAML: %v", err)
	}

	// Verify the exec provider is included
	if len(config.AuthInfos) != 1 {
		t.Fatalf("Expected 1 auth info, got %d", len(config.AuthInfos))
	}

	authInfo := config.AuthInfos[0]
	if authInfo.Name != "user" {
		t.Errorf("Expected auth info name to be 'user', got '%s'", authInfo.Name)
	}

	if authInfo.AuthInfo.Exec == nil {
		t.Fatalf("Expected exec provider to be present, got nil")
	}

	execConfig := authInfo.AuthInfo.Exec
	if execConfig.Command != "aws" {
		t.Errorf("Expected exec command to be 'aws', got '%s'", execConfig.Command)
	}

	if len(execConfig.Args) != 4 || execConfig.Args[0] != "eks" || execConfig.Args[1] != "get-token" {
		t.Errorf("Expected exec args to be ['eks', 'get-token', '--cluster-name', 'test-cluster'], got %v", execConfig.Args)
	}

	if len(execConfig.Env) != 1 || execConfig.Env[0].Name != "AWS_PROFILE" || execConfig.Env[0].Value != "test-profile" {
		t.Errorf("Expected exec env to have AWS_PROFILE=test-profile, got %v", execConfig.Env)
	}

	if execConfig.APIVersion != "client.authentication.k8s.io/v1beta1" {
		t.Errorf("Expected exec APIVersion to be 'client.authentication.k8s.io/v1beta1', got '%s'", execConfig.APIVersion)
	}
}
