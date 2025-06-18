package cmd

import (
	"context"
	"flag"
	"fmt"
	"regexp"
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
	portNameRe = regexp.MustCompile(`^:\d+$`)
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
	LogLevel           int
	Port               string
	MCPType            string
	SSEBaseUrl         string
	Kubeconfig         string
	Profile            string
	ListOutput         string
	ReadOnly           bool
	DisableDestructive bool

	profileObj    mcp.Profile
	listOutputObj output.Output

	genericiooptions.IOStreams
}

func NewMCPServerOptions(streams genericiooptions.IOStreams) *MCPServerOptions {
	return &MCPServerOptions{
		IOStreams:  streams,
		Profile:    "full",
		ListOutput: "table",
		MCPType:    "streamable",
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

	cmd.Flags().IntVar(&o.LogLevel, "log-level", o.LogLevel, "Set the log level (from 0 to 9)")
	cmd.Flags().StringVar(&o.Port, "port", o.Port, "Specify port in ':[PORT_NAME]' format to be used in sse or streamable HTTP servers")
	cmd.Flags().StringVar(&o.MCPType, "type", o.MCPType, "Transport type of the MCP Server. Options are sse, streamable, stdio. Default is streamable.")
	cmd.Flags().StringVar(&o.SSEBaseUrl, "sse-base-url", o.SSEBaseUrl, "SSE public base URL to use when sending the endpoint message (e.g. https://example.com)")
	cmd.Flags().StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "Path to the kubeconfig file to use for authentication. Only used in stdio.")
	cmd.Flags().StringVar(&o.Profile, "profile", o.Profile, "MCP profile to use (one of: "+strings.Join(mcp.ProfileNames, ", ")+")")
	cmd.Flags().StringVar(&o.ListOutput, "list-output", o.ListOutput, "Output format for resource list operations (one of: "+strings.Join(output.Names, ", ")+")")
	cmd.Flags().BoolVar(&o.ReadOnly, "read-only", o.ReadOnly, "If true, only tools annotated with readOnlyHint=true are exposed")
	cmd.Flags().BoolVar(&o.DisableDestructive, "disable-destructive", o.DisableDestructive, "If true, tools annotated with destructiveHint=true are disabled")

	cmd.AddCommand(version.NewCmdVersion(streams))
	return cmd
}

func (m *MCPServerOptions) Complete() error {
	m.initializeLogging()

	profile := mcp.ProfileFromString(m.Profile)
	if profile == nil {
		return fmt.Errorf("Invalid profile name: %s, valid names are: %s\n", m.Profile, strings.Join(mcp.ProfileNames, ", "))
	}
	m.profileObj = profile
	listOutput := output.FromString(m.ListOutput)
	if listOutput == nil {
		return fmt.Errorf("Invalid output name: %s, valid names are: %s\n", m.ListOutput, strings.Join(output.Names, ", "))
	}
	m.listOutputObj = listOutput

	return nil
}

func (m *MCPServerOptions) initializeLogging() {
	flagSet := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(flagSet)
	loggerOptions := []textlogger.ConfigOption{textlogger.Output(m.Out)}
	if m.LogLevel >= 0 {
		loggerOptions = append(loggerOptions, textlogger.Verbosity(m.LogLevel))
		_ = flagSet.Parse([]string{"--v", strconv.Itoa(m.LogLevel)})
	}
	logger := textlogger.NewLogger(textlogger.NewConfig(loggerOptions...))
	klog.SetLoggerWithOptions(logger)
}

func (m *MCPServerOptions) Validate() error {
	if !portNameRe.MatchString(m.Port) {
		return fmt.Errorf("invalid port name: %s", m.Port)
	}
	return nil
}

func (m *MCPServerOptions) Run() error {
	klog.V(1).Info("Starting kubernetes-mcp-server")
	klog.V(1).Infof(" - Profile: %s", m.profileObj.GetName())
	klog.V(1).Infof(" - ListOutput: %s", m.listOutputObj.GetName())
	klog.V(1).Infof(" - Read-only mode: %t", m.ReadOnly)
	klog.V(1).Infof(" - Disable destructive tools: %t", m.DisableDestructive)

	switch m.MCPType {
	case "streamable":
		mcpServer, err := mcp.NewServer(mcp.Configuration{
			Profile:            m.profileObj,
			ListOutput:         m.listOutputObj,
			ReadOnly:           m.ReadOnly,
			DisableDestructive: m.DisableDestructive,
		})
		if err != nil {
			return fmt.Errorf("Failed to initialize MCP server: %w\n", err)
		}
		defer mcpServer.Close()

		httpServer := mcpServer.ServeHTTP()
		klog.V(0).Infof("Streaming HTTP server starting on port %s and path /mcp", m.Port)
		if err := httpServer.Start(m.Port); err != nil {
			return fmt.Errorf("failed to start streaming HTTP server: %w\n", err)
		}
	case "sse":
		mcpServer, err := mcp.NewServer(mcp.Configuration{
			Profile:            m.profileObj,
			ListOutput:         m.listOutputObj,
			ReadOnly:           m.ReadOnly,
			DisableDestructive: m.DisableDestructive,
		})
		if err != nil {
			return fmt.Errorf("Failed to initialize MCP server: %w\n", err)
		}
		defer mcpServer.Close()

		ctx := context.Background()

		sseServer := mcpServer.ServeSse(m.SSEBaseUrl)
		defer func() { _ = sseServer.Shutdown(ctx) }()
		klog.V(0).Infof("SSE server starting on port %s and path /sse", m.Port)
		if err := sseServer.Start(m.Port); err != nil {
			return fmt.Errorf("failed to start SSE server: %w\n", err)
		}
	case "stdio":
		mcpServer, err := mcp.NewServer(mcp.Configuration{
			Profile:            m.profileObj,
			ListOutput:         m.listOutputObj,
			ReadOnly:           m.ReadOnly,
			DisableDestructive: m.DisableDestructive,
			Kubeconfig:         m.Kubeconfig,
		})
		if err != nil {
			return fmt.Errorf("Failed to initialize MCP server: %w\n", err)
		}
		defer mcpServer.Close()

		if err := mcpServer.ServeStdio(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid profile type: %s", m.Profile)
	}

	return nil
}
