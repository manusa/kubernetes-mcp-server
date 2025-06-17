package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/manusa/kubernetes-mcp-server/pkg/mcp"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
	"github.com/manusa/kubernetes-mcp-server/pkg/version"
)

var (
	long     = templates.LongDesc(i18n.T("Kubernetes Model Context Protocol (MCP) server"))
	examples = templates.Examples(i18n.T(`
# show this help
kubernetes-mcp-server -h

# shows version information
kubernetes-mcp-server --version

# start STDIO server
kubernetes-mcp-server

# start a SSE server on port 8080
kubernetes-mcp-server --sse-port 8080

# start a SSE server on port 8443 with a public HTTPS host of example.com
kubernetes-mcp-server --sse-port 8443 --sse-base-url https://example.com:8443
`))
)

type MCPServerOptions struct {
	Version            bool
	LogLevel           int
	SSEPort            int
	HttpPort           int
	SSEBaseUrl         string
	Kubeconfig         string
	Profile            string
	ListOutput         string
	ReadOnly           bool
	DisableDestructive bool

	genericiooptions.IOStreams
}

func NewMCPServerOptions(streams genericiooptions.IOStreams) *MCPServerOptions {
	return &MCPServerOptions{
		IOStreams:  streams,
		Profile:    "full",
		ListOutput: "table",
	}
}

func NewMCPServer(streams genericiooptions.IOStreams) *cobra.Command {
	o := NewMCPServerOptions(streams)
	cmd := &cobra.Command{
		Use:     "kubernetes-mcp-server [command] [options]",
		Short:   "Kubernetes Model Context Protocol (MCP) server",
		Long:    long,
		Example: examples,
		RunE: func(c *cobra.Command, args []string) error {
			if err := o.Complete(); err != nil {
				return err
			}
			if err := o.Validate(); err != nil {
				return err
			}
			if err := o.Run(); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&o.Version, "version", o.Version, "Print version information and quit")
	cmd.Flags().IntVar(&o.LogLevel, "log-level", o.LogLevel, "Set the log level (from 0 to 9)")
	cmd.Flags().IntVar(&o.SSEPort, "sse-port", o.SSEPort, "Start a SSE server on the specified port")
	cmd.Flags().IntVar(&o.HttpPort, "http-port", o.HttpPort, "Start a streamable HTTP server on the specified port")
	cmd.Flags().StringVar(&o.SSEBaseUrl, "sse-base-url", o.SSEBaseUrl, "SSE public base URL to use when sending the endpoint message (e.g. https://example.com)")
	cmd.Flags().StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "Path to the kubeconfig file to use for authentication")
	cmd.Flags().StringVar(&o.Profile, "profile", o.Profile, "MCP profile to use (one of: "+strings.Join(mcp.ProfileNames, ", ")+")")
	cmd.Flags().StringVar(&o.ListOutput, "list-output", o.ListOutput, "Output format for resource list operations (one of: "+strings.Join(output.Names, ", ")+")")
	cmd.Flags().BoolVar(&o.ReadOnly, "read-only", o.ReadOnly, "If true, only tools annotated with readOnlyHint=true are exposed")
	cmd.Flags().BoolVar(&o.DisableDestructive, "disable-destructive", o.DisableDestructive, "If true, tools annotated with destructiveHint=true are disabled")

	return cmd
}

func (m *MCPServerOptions) Complete() error {
	m.initializeLogging()

	return nil
}

func (m *MCPServerOptions) initializeLogging() {
	flagSet := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(flagSet)
	loggerOptions := []textlogger.ConfigOption{textlogger.Output(os.Stdout)}
	if m.LogLevel >= 0 {
		loggerOptions = append(loggerOptions, textlogger.Verbosity(m.LogLevel))
		_ = flagSet.Parse([]string{"--v", strconv.Itoa(m.LogLevel)})
	}
	logger := textlogger.NewLogger(textlogger.NewConfig(loggerOptions...))
	klog.SetLoggerWithOptions(logger)
}

func (m *MCPServerOptions) Validate() error {
	return nil
}

func (m *MCPServerOptions) Run() error {
	profile := mcp.ProfileFromString(m.Profile)
	if profile == nil {
		return fmt.Errorf("Invalid profile name: %s, valid names are: %s\n", m.Profile, strings.Join(mcp.ProfileNames, ", "))
	}
	listOutput := output.FromString(m.ListOutput)
	if listOutput == nil {
		return fmt.Errorf("Invalid output name: %s, valid names are: %s\n", m.ListOutput, strings.Join(output.Names, ", "))
	}
	klog.V(1).Info("Starting kubernetes-mcp-server")
	klog.V(1).Infof(" - Profile: %s", profile.GetName())
	klog.V(1).Infof(" - ListOutput: %s", listOutput.GetName())
	klog.V(1).Infof(" - Read-only mode: %t", m.ReadOnly)
	klog.V(1).Infof(" - Disable destructive tools: %t", m.DisableDestructive)

	if m.Version {
		fmt.Fprintf(m.Out, "%s\n", version.Version)
		return nil
	}
	mcpServer, err := mcp.NewSever(mcp.Configuration{
		Profile:            profile,
		ListOutput:         listOutput,
		ReadOnly:           m.ReadOnly,
		DisableDestructive: m.DisableDestructive,
		Kubeconfig:         m.Kubeconfig,
	})
	if err != nil {
		return fmt.Errorf("Failed to initialize MCP server: %w\n", err)
	}
	defer mcpServer.Close()

	ctx := context.Background()

	if m.SSEPort > 0 {
		sseServer := mcpServer.ServeSse(m.SSEBaseUrl)
		defer func() { _ = sseServer.Shutdown(ctx) }()
		klog.V(0).Infof("SSE server starting on port %d and path /sse", m.SSEPort)
		if err := sseServer.Start(fmt.Sprintf(":%d", m.SSEPort)); err != nil {
			return fmt.Errorf("failed to start SSE server: %w\n", err)
		}
	}

	if m.HttpPort > 0 {
		httpServer := mcpServer.ServeHTTP()
		klog.V(0).Infof("Streaming HTTP server starting on port %d and path /mcp", m.HttpPort)
		if err := httpServer.Start(fmt.Sprintf(":%d", m.HttpPort)); err != nil {
			return fmt.Errorf("failed to start streaming HTTP server: %w\n", err)
		}
	}

	if err := mcpServer.ServeStdio(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
