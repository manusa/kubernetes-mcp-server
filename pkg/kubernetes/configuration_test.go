package kubernetes

import (
	"errors"
	"k8s.io/client-go/rest"
	"os"
	"path"
	"runtime"
	"strings"
	"testing"
)

func TestKubernetes_IsInCluster(t *testing.T) {
	t.Run("with explicit kubeconfig", func(t *testing.T) {
		m := Manager{
			Kubeconfig: "kubeconfig",
		}
		if m.IsInCluster() {
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
		m := Manager{
			Kubeconfig: "",
		}
		if !m.IsInCluster() {
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
		m := Manager{
			Kubeconfig: "",
		}
		if m.IsInCluster() {
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
		m := Manager{
			Kubeconfig: "",
		}
		if m.IsInCluster() {
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
		m := Manager{Kubeconfig: path.Join(tempDir, "config")}
		err := resolveKubernetesConfigurations(&m)
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
		m := Manager{Kubeconfig: kubeconfigPath}
		err := resolveKubernetesConfigurations(&m)
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
		m := Manager{Kubeconfig: kubeconfigPath}
		err := resolveKubernetesConfigurations(&m)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if m.cfg == nil {
			t.Errorf("expected non-nil config, got nil")
		}
		if m.cfg.Host != "https://example.com" {
			t.Errorf("expected host https://example.com, got %s", m.cfg.Host)
		}
	})
}
