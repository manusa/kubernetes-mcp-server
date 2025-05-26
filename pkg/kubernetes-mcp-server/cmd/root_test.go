package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
)

func captureOutput(f func() error) (string, error) {
	originalOut := os.Stdout
	defer func() {
		os.Stdout = originalOut
	}()
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := f()
	_ = w.Close()
	out, _ := io.ReadAll(r)
	return string(out), err
}

func TestVersion(t *testing.T) {
	rootCmd.SetArgs([]string{"--version"})
	version, err := captureOutput(rootCmd.Execute)
	if version != "0.0.0\n" {
		t.Fatalf("Expected version 0.0.0, got %s %v", version, err)
	}
}

func TestDefaultProfile(t *testing.T) {
	rootCmd.SetArgs([]string{"--version", "--log-level=1"})
	out, err := captureOutput(rootCmd.Execute)
	if !strings.Contains(out, "- Profile: full") {
		t.Fatalf("Expected profile 'full', got %s %v", out, err)
	}
}

func TestDefaultReadOnly(t *testing.T) {
	rootCmd.SetArgs([]string{"--version", "--log-level=1"})
	out, err := captureOutput(rootCmd.Execute)
	if !strings.Contains(out, " - Read-only mode: false") {
		t.Fatalf("Expected read-only mode false, got %s %v", out, err)
	}
}

func TestDefaultDisableDestructive(t *testing.T) {
	rootCmd.SetArgs([]string{"--version", "--log-level=1"})
	out, err := captureOutput(rootCmd.Execute)
	if !strings.Contains(out, " - Disable destructive tools: false") {
		t.Fatalf("Expected disable destructive false, got %s %v", out, err)
	}
}
