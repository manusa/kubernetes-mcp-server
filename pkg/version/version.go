package version

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/kubectl/pkg/util/i18n"
)

var (
	CommitHash string
	BuildTime  string
	Version    = "0.0.0"
	BinaryName = "kubernetes-mcp-server"
)

type VersionOptions struct {
	Short bool

	genericiooptions.IOStreams
}

func NewVersionOptions(streams genericiooptions.IOStreams) *VersionOptions {
	return &VersionOptions{
		IOStreams: streams,
	}
}

func NewCmdVersion(streams genericiooptions.IOStreams) *cobra.Command {
	versionOptions := NewVersionOptions(streams)
	cmd := &cobra.Command{
		Use:   "version",
		Short: i18n.T("Print the version information"),
		Long:  "Print the version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if versionOptions.Short {
				err := versionOptions.PrintShortVersion()
				if err != nil {
					return err
				}
			} else {
				err := versionOptions.PrintVersion()
				if err != nil {
					return err
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&versionOptions.Short, "short", versionOptions.Short, "Print just the version number")
	return cmd
}

// PrintVersion prints the version details to the console.
func (v *VersionOptions) PrintVersion() error {
	fmt.Fprintf(v.Out, "Version: %s\n", Version)
	fmt.Fprintf(v.Out, "Commit: %s\n", CommitHash)
	fmt.Fprintf(v.Out, "Build Date: %s\n", BuildTime)
	fmt.Fprintf(v.Out, "Binary Name: %s\n", BinaryName)
	return nil
}

func (v *VersionOptions) PrintShortVersion() error {
	fmt.Fprintf(v.Out, "%s\n", Version)
	return nil
}
