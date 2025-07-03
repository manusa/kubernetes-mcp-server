package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"k8s.io/cli-runtime/pkg/genericiooptions"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
	"k8s.io/kubectl/pkg/util/i18n"
	"k8s.io/kubectl/pkg/util/templates"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
	internalhttp "github.com/manusa/kubernetes-mcp-server/pkg/http"
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
kubernetes-mcp-server --port 8080

# start a SSE server on port 8443 with a public HTTPS host of example.com
kubernetes-mcp-server --port 8443 --sse-base-url https://example.com:8443
`))
)

type MCPServerOptions struct {
	Version            bool
	LogLevel           int
	Port               string
	SSEPort            int
	HttpPort           int
	SSEBaseUrl         string
	Kubeconfig         string
	Profile            string
	ListOutput         string
	ReadOnly           bool
	DisableDestructive bool

	ConfigPath   string
	StaticConfig *config.StaticConfig

	genericiooptions.IOStreams
}

func NewMCPServerOptions(streams genericiooptions.IOStreams) *MCPServerOptions {
	return &MCPServerOptions{
		IOStreams:    streams,
		Profile:      "full",
		ListOutput:   "table",
		StaticConfig: &config.StaticConfig{},
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
			if err := o.Complete(c); err != nil {
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
	cmd.Flags().StringVar(&o.ConfigPath, "config", o.ConfigPath, "Path of the config file. Each profile has its set of defaults.")
	cmd.Flags().IntVar(&o.SSEPort, "sse-port", o.SSEPort, "Start a SSE server on the specified port")
	cmd.Flag("sse-port").Deprecated = "Use --port instead"
	cmd.Flags().IntVar(&o.HttpPort, "http-port", o.HttpPort, "Start a streamable HTTP server on the specified port")
	cmd.Flag("http-port").Deprecated = "Use --port instead"
	cmd.Flags().StringVar(&o.Port, "port", o.Port, "Start a streamable HTTP and SSE HTTP server on the specified port (e.g. 8080)")
	cmd.Flags().StringVar(&o.SSEBaseUrl, "sse-base-url", o.SSEBaseUrl, "SSE public base URL to use when sending the endpoint message (e.g. https://example.com)")
	cmd.Flags().StringVar(&o.Kubeconfig, "kubeconfig", o.Kubeconfig, "Path to the kubeconfig file to use for authentication")
	cmd.Flags().StringVar(&o.Profile, "profile", o.Profile, "MCP profile to use (one of: "+strings.Join(mcp.ProfileNames, ", ")+")")
	cmd.Flags().StringVar(&o.ListOutput, "list-output", o.ListOutput, "Output format for resource list operations (one of: "+strings.Join(output.Names, ", ")+"). Defaults to table.")
	cmd.Flags().BoolVar(&o.ReadOnly, "read-only", o.ReadOnly, "If true, only tools annotated with readOnlyHint=true are exposed")
	cmd.Flags().BoolVar(&o.DisableDestructive, "disable-destructive", o.DisableDestructive, "If true, tools annotated with destructiveHint=true are disabled")

	return cmd
}

func (m *MCPServerOptions) Complete(cmd *cobra.Command) error {
	if m.ConfigPath != "" {
		cnf, err := config.ReadConfig(m.ConfigPath)
		if err != nil {
			return err
		}
		m.StaticConfig = cnf
	}

	m.loadFlags(cmd)

	m.initializeLogging()

	return nil
}

func (m *MCPServerOptions) loadFlags(cmd *cobra.Command) {
	if cmd.Flag("log-level").Changed {
		m.StaticConfig.LogLevel = m.LogLevel
	}
	if cmd.Flag("port").Changed {
		m.StaticConfig.Port = m.Port
	} else if cmd.Flag("sse-port").Changed {
		m.StaticConfig.Port = strconv.Itoa(m.SSEPort)
	} else if cmd.Flag("http-port").Changed {
		m.StaticConfig.Port = strconv.Itoa(m.HttpPort)
	}
	if cmd.Flag("sse-base-url").Changed {
		m.StaticConfig.SSEBaseURL = m.SSEBaseUrl
	}
	if cmd.Flag("kubeconfig").Changed {
		m.StaticConfig.KubeConfig = m.Kubeconfig
	}
	if cmd.Flag("list-output").Changed || m.StaticConfig.ListOutput == "" {
		m.StaticConfig.ListOutput = m.ListOutput
	}
	if cmd.Flag("read-only").Changed {
		m.StaticConfig.ReadOnly = m.ReadOnly
	}
	if cmd.Flag("disable-destructive").Changed {
		m.StaticConfig.DisableDestructive = m.DisableDestructive
	}
}

func (m *MCPServerOptions) initializeLogging() {
	flagSet := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(flagSet)
	loggerOptions := []textlogger.ConfigOption{textlogger.Output(m.Out)}
	if m.StaticConfig.LogLevel >= 0 {
		loggerOptions = append(loggerOptions, textlogger.Verbosity(m.StaticConfig.LogLevel))
		_ = flagSet.Parse([]string{"--v", strconv.Itoa(m.StaticConfig.LogLevel)})
	}
	logger := textlogger.NewLogger(textlogger.NewConfig(loggerOptions...))
	klog.SetLoggerWithOptions(logger)
}

func (m *MCPServerOptions) Validate() error {
	if m.Port != "" && (m.SSEPort > 0 || m.HttpPort > 0) {
		return fmt.Errorf("--port is mutually exclusive with deprecated --http-port and --sse-port flags")
	}
	return nil
}

func (m *MCPServerOptions) Run() error {
	profile := mcp.ProfileFromString(m.Profile)
	if profile == nil {
		return fmt.Errorf("Invalid profile name: %s, valid names are: %s\n", m.Profile, strings.Join(mcp.ProfileNames, ", "))
	}
	listOutput := output.FromString(m.StaticConfig.ListOutput)
	if listOutput == nil {
		return fmt.Errorf("Invalid output name: %s, valid names are: %s\n", m.StaticConfig.ListOutput, strings.Join(output.Names, ", "))
	}
	klog.V(1).Info("Starting kubernetes-mcp-server")
	klog.V(1).Infof(" - Config: %s", m.ConfigPath)
	klog.V(1).Infof(" - Profile: %s", profile.GetName())
	klog.V(1).Infof(" - ListOutput: %s", listOutput.GetName())
	klog.V(1).Infof(" - Read-only mode: %t", m.StaticConfig.ReadOnly)
	klog.V(1).Infof(" - Disable destructive tools: %t", m.StaticConfig.DisableDestructive)

	if m.Version {
		_, _ = fmt.Fprintf(m.Out, "%s\n", version.Version)
		return nil
	}
	mcpServer, err := mcp.NewServer(mcp.Configuration{
		Profile:      profile,
		ListOutput:   listOutput,
		StaticConfig: m.StaticConfig,
	})
	if err != nil {
		return fmt.Errorf("Failed to initialize MCP server: %w\n", err)
	}
	defer mcpServer.Close()

	if m.StaticConfig.Port != "" {
		return internalhttp.Serve(mcpServer, m.StaticConfig.Port, m.SSEBaseUrl)
	}

	if err := mcpServer.ServeStdio(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}

	return nil
}
