package version

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"k8s.io/cli-runtime/pkg/genericiooptions"
)

func TestVersion(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := io.Discard
	rootCmd := NewVersionOptions(genericiooptions.IOStreams{In: in, Out: out, ErrOut: errOut})
	rootCmd.PrintVersion()
	if !strings.Contains(out.String(), "0.0.0") {
		t.Fatalf("Expected version 0.0.0, got %s", out.String())
	}
}

func TestVersionShort(t *testing.T) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := io.Discard
	rootCmd := NewVersionOptions(genericiooptions.IOStreams{In: in, Out: out, ErrOut: errOut})
	rootCmd.PrintShortVersion()
	if out.String() != "0.0.0\n" {
		t.Fatalf("Expected version 0.0.0, got %s", out.String())
	}
}
