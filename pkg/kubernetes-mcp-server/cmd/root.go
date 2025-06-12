package cmd

import (
	"errors"
	"flag"
	"fmt"
	"github.com/manusa/kubernetes-mcp-server/pkg/mcp"
	"github.com/manusa/kubernetes-mcp-server/pkg/output"
	"github.com/manusa/kubernetes-mcp-server/pkg/version"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"
	"os"
	"strconv"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "kubernetes-mcp-server [command] [options]",
	Short: "Kubernetes Model Context Protocol (MCP) server",
	Long: `
Kubernetes Model Context Protocol (MCP) server

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

  # TODO: add more examples`,
	Run: func(cmd *cobra.Command, args []string) {
		initLogging()
		profile := mcp.ProfileFromString(viper.GetString("profile"))
		if profile == nil {
			fmt.Printf("Invalid profile name: %s, valid names are: %s\n", viper.GetString("profile"), strings.Join(mcp.ProfileNames, ", "))
			os.Exit(1)
		}
		listOutput := output.FromString(viper.GetString("list-output"))
		if listOutput == nil {
			fmt.Printf("Invalid output name: %s, valid names are: %s\n", viper.GetString("list-output"), strings.Join(output.Names, ", "))
			os.Exit(1)
		}
		klog.V(1).Info("Starting kubernetes-mcp-server")
		klog.V(1).Infof(" - Profile: %s", profile.GetName())
		klog.V(1).Infof(" - ListOutput: %s", listOutput.GetName())
		klog.V(1).Infof(" - Read-only mode: %t", viper.GetBool("read-only"))
		klog.V(1).Infof(" - Disable destructive tools: %t", viper.GetBool("disable-destructive"))
		if viper.GetBool("version") {
			fmt.Println(version.Version)
			return
		}
		mcpServer, err := mcp.NewSever(mcp.Configuration{
			Profile:            profile,
			ListOutput:         listOutput,
			ReadOnly:           viper.GetBool("read-only"),
			DisableDestructive: viper.GetBool("disable-destructive"),
			Kubeconfig:         viper.GetString("kubeconfig"),
		})
		if err != nil {
			fmt.Printf("Failed to initialize MCP server: %v\n", err)
			os.Exit(1)
		}
		defer mcpServer.Close()

		var sseServer *server.SSEServer
		if ssePort := viper.GetInt("sse-port"); ssePort > 0 {
			sseServer = mcpServer.ServeSse(viper.GetString("sse-base-url"))
			defer func() { _ = sseServer.Shutdown(cmd.Context()) }()
			klog.V(0).Infof("SSE server starting on port %d", ssePort)
			if err := sseServer.Start(fmt.Sprintf(":%d", ssePort)); err != nil {
				klog.Errorf("Failed to start SSE server: %s", err)
				return
			}
		}
		if err := mcpServer.ServeStdio(); err != nil && !errors.Is(err, context.Canceled) {
			panic(err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		klog.Errorf("Failed to execute command: %s", err)
		os.Exit(1)
	}
}

func initLogging() {
	flagSet := flag.NewFlagSet("kubernetes-mcp-server", flag.ContinueOnError)
	klog.InitFlags(flagSet)
	loggerOptions := []textlogger.ConfigOption{textlogger.Output(os.Stdout)}
	if logLevel := viper.GetInt("log-level"); logLevel >= 0 {
		loggerOptions = append(loggerOptions, textlogger.Verbosity(logLevel))
		_ = flagSet.Parse([]string{"--v", strconv.Itoa(logLevel)})
	}
	logger := textlogger.NewLogger(textlogger.NewConfig(loggerOptions...))
	klog.SetLoggerWithOptions(logger)
}

// flagInit initializes the flags for the root command.
// Exposed for testing purposes.
func flagInit() {
	rootCmd.Flags().BoolP("version", "v", false, "Print version information and quit")
	rootCmd.Flags().IntP("log-level", "", 0, "Set the log level (from 0 to 9)")
	rootCmd.Flags().IntP("sse-port", "", 0, "Start a SSE server on the specified port")
	rootCmd.Flags().StringP("sse-base-url", "", "", "SSE public base URL to use when sending the endpoint message (e.g. https://example.com)")
	rootCmd.Flags().StringP("kubeconfig", "", "", "Path to the kubeconfig file to use for authentication")
	rootCmd.Flags().String("profile", "full", "MCP profile to use (one of: "+strings.Join(mcp.ProfileNames, ", ")+")")
	rootCmd.Flags().String("list-output", "table", "Output format for resource lists (one of: "+strings.Join(output.Names, ", ")+")")
	rootCmd.Flags().Bool("read-only", false, "If true, only tools annotated with readOnlyHint=true are exposed")
	rootCmd.Flags().Bool("disable-destructive", false, "If true, tools annotated with destructiveHint=true are disabled")
	_ = viper.BindPFlags(rootCmd.Flags())
}

func init() {
	flagInit()
}
