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

func TestProfile(t *testing.T) {
	t.Run("default", func(t *testing.T) {
		in := &bytes.Buffer{}
		out := &bytes.Buffer{}
		errOut := io.Discard
		rootCmd := NewMCPServerOptions(genericiooptions.IOStreams{In: in, Out: out, ErrOut: errOut})
		rootCmd.LogLevel = 1
		rootCmd.Complete()
		rootCmd.Run()
		if !strings.Contains(out.String(), "- Profile: full") {
			t.Fatalf("Expected profile 'full', got %s", out)
		}
	})
}

func TestListOutput(t *testing.T) {
	t.Run("available", func(t *testing.T) {
		in := &bytes.Buffer{}
		out := io.Discard
		errOut := io.Discard
		rootCmd := NewMCPServer(genericiooptions.IOStreams{In: in, Out: out, ErrOut: errOut})
		rootCmd.SetArgs([]string{"--help"})
		o, err := captureOutput(rootCmd.Execute)
		if !strings.Contains(o, "Output format for resource list operations (one of: yaml, table)") {
			t.Fatalf("Expected all available outputs, got %s %v", out, err)
		}
	})
	t.Run("defaults to table", func(t *testing.T) {
		in := &bytes.Buffer{}
		out := &bytes.Buffer{}
		errOut := io.Discard
		rootCmd := NewMCPServerOptions(genericiooptions.IOStreams{In: in, Out: out, ErrOut: errOut})
		rootCmd.LogLevel = 1
		rootCmd.Complete()
		rootCmd.Run()
		if !strings.Contains(out.String(), "- ListOutput: table") {
			t.Fatalf("Expected list-output 'table', got %s", out)
		}
	})
}

func TestDefaultReadOnly(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := io.Discard
	rootCmd := NewMCPServerOptions(genericiooptions.IOStreams{In: in, Out: out, ErrOut: errOut})
	rootCmd.LogLevel = 1
	rootCmd.Complete()
	rootCmd.Run()
	if !strings.Contains(out.String(), " - Read-only mode: false") {
		t.Fatalf("Expected read-only mode false, got %s", out)
	}
}

func TestDefaultDisableDestructive(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := io.Discard
	rootCmd := NewMCPServerOptions(genericiooptions.IOStreams{In: in, Out: out, ErrOut: errOut})
	rootCmd.LogLevel = 1
	rootCmd.Complete()
	rootCmd.Run()
	if !strings.Contains(out.String(), " - Disable destructive tools: false") {
		t.Fatalf("Expected disable destructive false, got %s", out)
	}
}
