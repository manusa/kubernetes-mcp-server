package http

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/textlogger"

	"github.com/manusa/kubernetes-mcp-server/pkg/config"
	"github.com/manusa/kubernetes-mcp-server/pkg/mcp"
)

type httpContext struct {
	t               *testing.T
	klogState       klog.State
	logBuffer       bytes.Buffer
	httpAddress     string             // HTTP server address
	timeoutCancel   context.CancelFunc // Release resources if test completes before the timeout
	stopServer      context.CancelFunc
	waitForShutdown func() error
}

func (c *httpContext) beforeEach() {
	http.DefaultClient.Timeout = 10 * time.Second
	// Fake Kubernetes configuration
	fakeConfig := api.NewConfig()
	fakeConfig.Clusters["fake"] = api.NewCluster()
	fakeConfig.Clusters["fake"].Server = "https://example.com"
	fakeConfig.Contexts["fake-context"] = api.NewContext()
	fakeConfig.Contexts["fake-context"].Cluster = "fake"
	fakeConfig.CurrentContext = "fake-context"
	kubeConfig := filepath.Join(c.t.TempDir(), "config")
	_ = clientcmd.WriteToFile(*fakeConfig, kubeConfig)
	_ = os.Setenv("KUBECONFIG", kubeConfig)
	// Capture logging
	c.klogState = klog.CaptureState()
	klog.SetLogger(textlogger.NewLogger(textlogger.NewConfig(textlogger.Verbosity(1), textlogger.Output(&c.logBuffer))))
	// Start server in random port
	ln, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		c.t.Fatalf("Failed to find random port for HTTP server: %v", err)
	}
	c.httpAddress = ln.Addr().String()
	if randomPortErr := ln.Close(); randomPortErr != nil {
		c.t.Fatalf("Failed to close random port listener: %v", randomPortErr)
	}
	staticConfig := &config.StaticConfig{Port: fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)}
	mcpServer, err := mcp.NewServer(mcp.Configuration{
		Profile:      mcp.Profiles[0],
		StaticConfig: staticConfig,
	})
	if err != nil {
		c.t.Fatalf("Failed to create MCP server: %v", err)
	}
	var timeoutCtx, cancelCtx context.Context
	timeoutCtx, c.timeoutCancel = context.WithTimeout(c.t.Context(), 10*time.Second)
	group, gc := errgroup.WithContext(timeoutCtx)
	cancelCtx, c.stopServer = context.WithCancel(gc)
	group.Go(func() error { return Serve(cancelCtx, mcpServer, staticConfig) })
	c.waitForShutdown = group.Wait
	// Wait for HTTP server to start (using net)
	for i := 0; i < 10; i++ {
		conn, err := net.Dial("tcp", c.httpAddress)
		if err == nil {
			_ = conn.Close()
			break
		}
		time.Sleep(50 * time.Millisecond) // Wait before retrying
	}
}

func (c *httpContext) afterEach() {
	c.stopServer()
	err := c.waitForShutdown()
	if err != nil {
		c.t.Errorf("HTTP server did not shut down gracefully: %v", err)
	}
	c.timeoutCancel()
	c.klogState.Restore()
	_ = os.Setenv("KUBECONFIG", "")
}

func testCase(t *testing.T, test func(c *httpContext)) {
	ctx := &httpContext{t: t}
	ctx.beforeEach()
	t.Cleanup(ctx.afterEach)
	test(ctx)
}

func TestGracefulShutdown(t *testing.T) {
	testCase(t, func(ctx *httpContext) {
		ctx.stopServer()
		err := ctx.waitForShutdown()
		t.Run("Stops gracefully", func(t *testing.T) {
			if err != nil {
				t.Errorf("Expected graceful shutdown, but got error: %v", err)
			}
		})
		t.Run("Stops on context cancel", func(t *testing.T) {
			if !strings.Contains(ctx.logBuffer.String(), "Context cancelled, initiating graceful shutdown") {
				t.Errorf("Context cancelled, initiating graceful shutdown, got: %s", ctx.logBuffer.String())
			}
		})
		t.Run("Starts server shutdown", func(t *testing.T) {
			if !strings.Contains(ctx.logBuffer.String(), "Shutting down HTTP server gracefully") {
				t.Errorf("Expected graceful shutdown log, got: %s", ctx.logBuffer.String())
			}
		})
		t.Run("Server shutdown completes", func(t *testing.T) {
			if !strings.Contains(ctx.logBuffer.String(), "HTTP server shutdown complete") {
				t.Errorf("Expected HTTP server shutdown completed log, got: %s", ctx.logBuffer.String())
			}
		})
	})
}

func TestSseTransport(t *testing.T) {
	testCase(t, func(ctx *httpContext) {
		sseResp, sseErr := http.Get(fmt.Sprintf("http://%s/sse", ctx.httpAddress))
		t.Cleanup(func() { _ = sseResp.Body.Close() })
		t.Run("Exposes SSE endpoint at /sse", func(t *testing.T) {
			if sseErr != nil {
				t.Fatalf("Failed to get SSE endpoint: %v", sseErr)
			}
			if sseResp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 OK, got %d", sseResp.StatusCode)
			}
		})
		t.Run("SSE endpoint returns text/event-stream content type", func(t *testing.T) {
			if sseResp.Header.Get("Content-Type") != "text/event-stream" {
				t.Errorf("Expected Content-Type text/event-stream, got %s", sseResp.Header.Get("Content-Type"))
			}
		})
		responseReader := bufio.NewReader(sseResp.Body)
		event, eventErr := responseReader.ReadString('\n')
		endpoint, endpointErr := responseReader.ReadString('\n')
		t.Run("SSE endpoint returns stream with messages endpoint", func(t *testing.T) {
			if eventErr != nil {
				t.Fatalf("Failed to read SSE response body (event): %v", eventErr)
			}
			if event != "event: endpoint\n" {
				t.Errorf("Expected SSE event 'endpoint', got %s", event)
			}
			if endpointErr != nil {
				t.Fatalf("Failed to read SSE response body (endpoint): %v", endpointErr)
			}
			if !strings.HasPrefix(endpoint, "data: /message?sessionId=") {
				t.Errorf("Expected SSE data: '/message', got %s", endpoint)
			}
		})
		messageResp, messageErr := http.Post(
			fmt.Sprintf("http://%s/message?sessionId=%s", ctx.httpAddress, strings.TrimSpace(endpoint[25:])),
			"application/json",
			bytes.NewBufferString("{}"),
		)
		t.Cleanup(func() { _ = messageResp.Body.Close() })
		t.Run("Exposes message endpoint at /message", func(t *testing.T) {
			if messageErr != nil {
				t.Fatalf("Failed to get message endpoint: %v", messageErr)
			}
			if messageResp.StatusCode != http.StatusAccepted {
				t.Errorf("Expected HTTP 202 OK, got %d", messageResp.StatusCode)
			}
		})
	})
}

func TestStreamableHttpTransport(t *testing.T) {
	testCase(t, func(ctx *httpContext) {
		mcpGetResp, mcpGetErr := http.Get(fmt.Sprintf("http://%s/mcp", ctx.httpAddress))
		t.Cleanup(func() { _ = mcpGetResp.Body.Close() })
		t.Run("Exposes MCP GET endpoint at /mcp", func(t *testing.T) {
			if mcpGetErr != nil {
				t.Fatalf("Failed to get MCP endpoint: %v", mcpGetErr)
			}
			if mcpGetResp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 OK, got %d", mcpGetResp.StatusCode)
			}
		})
		t.Run("MCP GET endpoint returns text/event-stream content type", func(t *testing.T) {
			if mcpGetResp.Header.Get("Content-Type") != "text/event-stream" {
				t.Errorf("Expected Content-Type text/event-stream (GET), got %s", mcpGetResp.Header.Get("Content-Type"))
			}
		})
		mcpPostResp, mcpPostErr := http.Post(fmt.Sprintf("http://%s/mcp", ctx.httpAddress), "application/json", bytes.NewBufferString("{}"))
		t.Cleanup(func() { _ = mcpPostResp.Body.Close() })
		t.Run("Exposes MCP POST endpoint at /mcp", func(t *testing.T) {
			if mcpPostErr != nil {
				t.Fatalf("Failed to post to MCP endpoint: %v", mcpPostErr)
			}
			if mcpPostResp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 OK, got %d", mcpPostResp.StatusCode)
			}
		})
		t.Run("MCP POST endpoint returns application/json content type", func(t *testing.T) {
			if mcpPostResp.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json (POST), got %s", mcpPostResp.Header.Get("Content-Type"))
			}
		})
	})
}

func TestHealthCheck(t *testing.T) {
	testCase(t, func(ctx *httpContext) {
		t.Run("Exposes health check endpoint at /healthz", func(t *testing.T) {
			resp, err := http.Get(fmt.Sprintf("http://%s/healthz", ctx.httpAddress))
			if err != nil {
				t.Fatalf("Failed to get health check endpoint: %v", err)
			}
			t.Cleanup(func() { _ = resp.Body.Close })
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 OK, got %d", resp.StatusCode)
			}
		})
	})
}

func TestWellKnownOAuthProtectedResource(t *testing.T) {
	testCase(t, func(ctx *httpContext) {
		resp, err := http.Get(fmt.Sprintf("http://%s/.well-known/oauth-protected-resource", ctx.httpAddress))
		t.Cleanup(func() { _ = resp.Body.Close() })
		t.Run("Exposes .well-known/oauth-protected-resource endpoint", func(t *testing.T) {
			if err != nil {
				t.Fatalf("Failed to get .well-known/oauth-protected-resource endpoint: %v", err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected HTTP 200 OK, got %d", resp.StatusCode)
			}
		})
		t.Run(".well-known/oauth-protected-resource returns application/json content type", func(t *testing.T) {
			if resp.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
			}
		})
	})
}
