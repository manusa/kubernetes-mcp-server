package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericiooptions"
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

func testStream() (genericiooptions.IOStreams, *bytes.Buffer) {
	out := &bytes.Buffer{}
	return genericiooptions.IOStreams{
		In:     &bytes.Buffer{},
		Out:    out,
		ErrOut: io.Discard,
	}, out
}

func TestVersion(t *testing.T) {
	ioStreams, out := testStream()
	rootCmd := NewMCPServer(ioStreams)
	rootCmd.SetArgs([]string{"--version"})
	if err := rootCmd.Execute(); out.String() != "0.0.0\n" {
		t.Fatalf("Expected version 0.0.0, got %s %v", out.String(), err)
	}
}

func TestProfile(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		ioStreams, _ := testStream()
		rootCmd := NewMCPServer(ioStreams)
		rootCmd.SetArgs([]string{"--help"})
		o, err := captureOutput(rootCmd.Execute) // --help doesn't use logger/klog, cobra prints directly to stdout
		if !strings.Contains(o, "MCP profile to use (one of: full) ") {
			t.Fatalf("Expected all available profiles, got %s %v", o, err)
		}
	})
	t.Run("default", func(t *testing.T) {
		ioStreams, out := testStream()
		rootCmd := NewMCPServer(ioStreams)
		rootCmd.SetArgs([]string{"--version", "--log-level=1"})
		if err := rootCmd.Execute(); !strings.Contains(out.String(), "- Profile: full") {
			t.Fatalf("Expected profile 'full', got %s %v", out, err)
		}
	})
}

func TestListOutput(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		ioStreams, _ := testStream()
		rootCmd := NewMCPServer(ioStreams)
		rootCmd.SetArgs([]string{"--help"})
		o, err := captureOutput(rootCmd.Execute) // --help doesn't use logger/klog, cobra prints directly to stdout
		if !strings.Contains(o, "Output format for resource list operations (one of: yaml, table)") {
			t.Fatalf("Expected all available outputs, got %s %v", o, err)
		}
	})
	t.Run("defaults to table", func(t *testing.T) {
		ioStreams, out := testStream()
		rootCmd := NewMCPServer(ioStreams)
		rootCmd.SetArgs([]string{"--version", "--log-level=1"})
		if err := rootCmd.Execute(); !strings.Contains(out.String(), "- ListOutput: table") {
			t.Fatalf("Expected list-output 'table', got %s %v", out, err)
		}
	})
}

func TestReadOnly(t *testing.T) {
	t.Run("defaults to false", func(t *testing.T) {
		ioStreams, out := testStream()
		rootCmd := NewMCPServer(ioStreams)
		rootCmd.SetArgs([]string{"--version", "--log-level=1"})
		if err := rootCmd.Execute(); !strings.Contains(out.String(), " - Read-only mode: false") {
			t.Fatalf("Expected read-only mode false, got %s %v", out, err)
		}
	})
}

func TestDisableDestructive(t *testing.T) {
	t.Run("defaults to false", func(t *testing.T) {
		ioStreams, out := testStream()
		rootCmd := NewMCPServer(ioStreams)
		rootCmd.SetArgs([]string{"--version", "--log-level=1"})
		if err := rootCmd.Execute(); !strings.Contains(out.String(), " - Disable destructive tools: false") {
			t.Fatalf("Expected disable destructive false, got %s %v", out, err)
		}
	})
}
