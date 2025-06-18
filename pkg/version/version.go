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

func NewCmdVersion(ioStreams genericiooptions.IOStreams) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: i18n.T("Print the version information"),
		Long:  "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			PrintVersion()
		},
	}
	return cmd
}

// PrintVersion prints the version details to the console.
func PrintVersion() {
	fmt.Printf("Version: %s\n", Version)
	fmt.Printf("Commit: %s\n", CommitHash)
	fmt.Printf("Build Date: %s\n", BuildTime)
	fmt.Printf("Binary Name: %s\n", BinaryName)
}
