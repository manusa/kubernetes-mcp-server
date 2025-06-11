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
	rootCmd.ResetFlags()
	flagInit()
	version, err := captureOutput(rootCmd.Execute)
	if version != "0.0.0\n" {
		t.Fatalf("Expected version 0.0.0, got %s %v", version, err)
	}
}

func TestProfile(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		rootCmd.SetArgs([]string{"--version", "--log-level=1"})
		rootCmd.ResetFlags()
		flagInit()
		out, err := captureOutput(rootCmd.Execute)
		if !strings.Contains(out, "- Profile: full") {
			t.Fatalf("Expected profile 'full', got %s %v", out, err)
		}
	})
}

func TestOutput(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		rootCmd.SetArgs([]string{"--help"})
		rootCmd.ResetFlags()
		flagInit()
		out, err := captureOutput(rootCmd.Execute)
		if !strings.Contains(out, "Output format for resources (one of: yaml)") {
			t.Fatalf("Expected all available outputs, got %s %v", out, err)
		}
	})
	t.Run("default", func(t *testing.T) {
		rootCmd.SetArgs([]string{"--version", "--log-level=1"})
		rootCmd.ResetFlags()
		flagInit()
		out, err := captureOutput(rootCmd.Execute)
		if !strings.Contains(out, "- Output: yaml") {
			t.Fatalf("Expected output 'yaml', got %s %v", out, err)
		}
	})
}

func TestDefaultReadOnly(t *testing.T) {
	rootCmd.SetArgs([]string{"--version", "--log-level=1"})
	rootCmd.ResetFlags()
	flagInit()
	out, err := captureOutput(rootCmd.Execute)
	if !strings.Contains(out, " - Read-only mode: false") {
		t.Fatalf("Expected read-only mode false, got %s %v", out, err)
	}
}

func TestDefaultDisableDestructive(t *testing.T) {
	rootCmd.SetArgs([]string{"--version", "--log-level=1"})
	rootCmd.ResetFlags()
	flagInit()
	out, err := captureOutput(rootCmd.Execute)
	if !strings.Contains(out, " - Disable destructive tools: false") {
		t.Fatalf("Expected disable destructive false, got %s %v", out, err)
	}
}
