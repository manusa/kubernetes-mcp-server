package kubernetes

import (
	"context"
	"testing"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func TestDerived(t *testing.T) {
	t.Run("with bearer token", func(t *testing.T) {
		// Setup
		mockConfig := &api.Config{
			Clusters: map[string]*api.Cluster{
				"test-cluster": {
					Server: "https://example.com",
				},
			},
			AuthInfos: map[string]*api.AuthInfo{
				"test-user": {
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
				},
			},
			Contexts: map[string]*api.Context{
				"test-context": {
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				},
			},
			CurrentContext: "test-context",
		}

		k := &Kubernetes{
			cfg: &rest.Config{
				Host:        "https://example.com",
				BearerToken: "original-token",
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
				ExecProvider: &api.ExecConfig{
					Command: "test-command",
				},
			},
			clientCmdConfig: clientcmd.NewDefaultClientConfig(*mockConfig, &clientcmd.ConfigOverrides{}),
		}

		// Create context with bearer token
		ctx := context.WithValue(context.Background(), AuthorizationBearerTokenHeader, "new-token")

		// Execute
		derived := k.Derived(ctx)

		// Verify
		if derived == k {
			t.Error("expected a new derived instance, got the same instance")
		}
		if derived.cfg.BearerToken != "new-token" {
			t.Errorf("expected bearer token to be 'new-token', got '%s'", derived.cfg.BearerToken)
		}
		if derived.cfg.AuthProvider != nil {
			t.Errorf("expected AuthProvider to be nil, got %v", derived.cfg.AuthProvider)
		}
		if derived.cfg.ExecProvider != nil {
			t.Errorf("expected ExecProvider to be nil, got %v", derived.cfg.ExecProvider)
		}
		if derived.cfg.BearerTokenFile != "" {
			t.Errorf("expected BearerTokenFile to be empty, got '%s'", derived.cfg.BearerTokenFile)
		}
		if derived.cfg.Username != "" {
			t.Errorf("expected Username to be empty, got '%s'", derived.cfg.Username)
		}
		if derived.cfg.Password != "" {
			t.Errorf("expected Password to be empty, got '%s'", derived.cfg.Password)
		}
	})

	t.Run("without bearer token", func(t *testing.T) {
		// Setup
		mockConfig := &api.Config{
			Clusters: map[string]*api.Cluster{
				"test-cluster": {
					Server: "https://example.com",
				},
			},
			AuthInfos: map[string]*api.AuthInfo{
				"test-user": {
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
				},
			},
			Contexts: map[string]*api.Context{
				"test-context": {
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				},
			},
			CurrentContext: "test-context",
		}

		k := &Kubernetes{
			cfg: &rest.Config{
				Host:        "https://example.com",
				BearerToken: "original-token",
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
				ExecProvider: &api.ExecConfig{
					Command: "test-command",
				},
			},
			clientCmdConfig: clientcmd.NewDefaultClientConfig(*mockConfig, &clientcmd.ConfigOverrides{}),
		}

		// Create context without bearer token
		ctx := context.Background()

		// Execute
		derived := k.Derived(ctx)

		// Verify - should return the same instance when no bearer token
		if derived != k {
			t.Error("expected the same instance when no bearer token is provided")
		}
	})

	t.Run("with empty bearer token", func(t *testing.T) {
		// Setup
		mockConfig := &api.Config{
			Clusters: map[string]*api.Cluster{
				"test-cluster": {
					Server: "https://example.com",
				},
			},
			AuthInfos: map[string]*api.AuthInfo{
				"test-user": {
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
				},
			},
			Contexts: map[string]*api.Context{
				"test-context": {
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				},
			},
			CurrentContext: "test-context",
		}

		k := &Kubernetes{
			cfg: &rest.Config{
				Host:        "https://example.com",
				BearerToken: "original-token",
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
				ExecProvider: &api.ExecConfig{
					Command: "test-command",
				},
			},
			clientCmdConfig: clientcmd.NewDefaultClientConfig(*mockConfig, &clientcmd.ConfigOverrides{}),
		}

		// Create context with empty bearer token
		ctx := context.WithValue(context.Background(), AuthorizationBearerTokenHeader, "")

		// Execute
		derived := k.Derived(ctx)

		// Verify - should return the same instance when empty bearer token is provided
		if derived != k {
			t.Error("expected the same instance when empty bearer token is provided")
		}
	})

	t.Run("OIDC configuration preservation", func(t *testing.T) {
		// Setup
		mockConfig := &api.Config{
			Clusters: map[string]*api.Cluster{
				"test-cluster": {
					Server: "https://example.com",
				},
			},
			AuthInfos: map[string]*api.AuthInfo{
				"test-user": {
					AuthProvider: &api.AuthProviderConfig{
						Name: "oidc",
						Config: map[string]string{
							"client-id":      "test-client",
							"client-secret":  "test-secret",
							"id-token":       "test-id-token",
							"refresh-token":  "test-refresh-token",
							"idp-issuer-url": "https://oidc.example.org",
						},
					},
				},
			},
			Contexts: map[string]*api.Context{
				"test-context": {
					Cluster:  "test-cluster",
					AuthInfo: "test-user",
				},
			},
			CurrentContext: "test-context",
		}

		k := &Kubernetes{
			cfg: &rest.Config{
				Host: "https://example.com",
				AuthProvider: &api.AuthProviderConfig{
					Name: "oidc",
					Config: map[string]string{
						"client-id":      "test-client",
						"client-secret":  "test-secret",
						"id-token":       "test-id-token",
						"refresh-token":  "test-refresh-token",
						"idp-issuer-url": "https://oidc.example.org",
					},
				},
			},
			clientCmdConfig: clientcmd.NewDefaultClientConfig(*mockConfig, &clientcmd.ConfigOverrides{}),
		}

		// Create context without bearer token
		ctx := context.Background()

		// Execute
		derived := k.Derived(ctx)

		// Verify - should return same instance and preserve OIDC config
		if derived != k {
			t.Error("expected the same instance when no bearer token is provided")
		}
		if k.cfg.AuthProvider == nil {
			t.Error("expected original OIDC AuthProvider to be preserved")
		}
		if k.cfg.AuthProvider.Name != "oidc" {
			t.Errorf("expected AuthProvider.Name to be 'oidc', got '%s'", k.cfg.AuthProvider.Name)
		}
		if k.cfg.AuthProvider.Config["idp-issuer-url"] != "https://oidc.example.org" {
			t.Errorf("expected idp-issuer-url to be preserved, got '%s'", k.cfg.AuthProvider.Config["idp-issuer-url"])
		}
	})
}

func TestConfigurationViewOIDC(t *testing.T) {
	t.Run("in-cluster with OIDC auth provider", func(t *testing.T) {
		// Mock InClusterConfig to return a valid config
		originalInClusterConfig := InClusterConfig
		defer func() { InClusterConfig = originalInClusterConfig }()

		InClusterConfig = func() (*rest.Config, error) {
			return &rest.Config{
				Host: "https://kubernetes.default.svc",
				AuthProvider: &api.AuthProviderConfig{
					Name: "oidc",
					Config: map[string]string{
						"client-id":      "test-client",
						"idp-issuer-url": "https://oidc.example.org",
					},
				},
			}, nil
		}

		k := &Kubernetes{
			cfg: &rest.Config{
				Host: "https://kubernetes.default.svc",
				AuthProvider: &api.AuthProviderConfig{
					Name: "oidc",
					Config: map[string]string{
						"client-id":      "test-client",
						"idp-issuer-url": "https://oidc.example.org",
					},
				},
			},
		}

		// Execute
		configYaml, err := k.ConfigurationView(false)

		// Verify
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if configYaml == "" {
			t.Error("expected non-empty configuration YAML")
		}

		// Check that OIDC configuration is included
		if !contains(configYaml, "oidc") {
			t.Error("expected OIDC configuration to be present in the output")
		}
		if !contains(configYaml, "test-client") {
			t.Error("expected client-id to be present in the output")
		}
		if !contains(configYaml, "https://oidc.example.org") {
			t.Error("expected idp-issuer-url to be present in the output")
		}
	})

	t.Run("in-cluster with exec provider", func(t *testing.T) {
		// Mock InClusterConfig to return a valid config
		originalInClusterConfig := InClusterConfig
		defer func() { InClusterConfig = originalInClusterConfig }()

		InClusterConfig = func() (*rest.Config, error) {
			return &rest.Config{
				Host: "https://kubernetes.default.svc",
				ExecProvider: &api.ExecConfig{
					Command: "oidc-login",
					Args:    []string{"get-token"},
				},
			}, nil
		}

		k := &Kubernetes{
			cfg: &rest.Config{
				Host: "https://kubernetes.default.svc",
				ExecProvider: &api.ExecConfig{
					Command: "oidc-login",
					Args:    []string{"get-token"},
				},
			},
		}

		// Execute
		configYaml, err := k.ConfigurationView(false)

		// Verify
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if configYaml == "" {
			t.Error("expected non-empty configuration YAML")
		}

		// Check that exec configuration is included
		if !contains(configYaml, "oidc-login") {
			t.Error("expected exec command to be present in the output")
		}
		if !contains(configYaml, "get-token") {
			t.Error("expected exec args to be present in the output")
		}
	})

	t.Run("in-cluster with bearer token", func(t *testing.T) {
		// Mock InClusterConfig to return a valid config
		originalInClusterConfig := InClusterConfig
		defer func() { InClusterConfig = originalInClusterConfig }()

		InClusterConfig = func() (*rest.Config, error) {
			return &rest.Config{
				Host:        "https://kubernetes.default.svc",
				BearerToken: "test-bearer-token",
			}, nil
		}

		k := &Kubernetes{
			cfg: &rest.Config{
				Host:        "https://kubernetes.default.svc",
				BearerToken: "test-bearer-token",
			},
		}

		// Execute
		configYaml, err := k.ConfigurationView(false)

		// Verify
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if configYaml == "" {
			t.Error("expected non-empty configuration YAML")
		}

		// Check that token configuration is included
		if !contains(configYaml, "test-bearer-token") {
			t.Error("expected bearer token to be present in the output")
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
