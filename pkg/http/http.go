package http

import (
	"errors"
	"net/http"

	"k8s.io/klog/v2"

	"github.com/manusa/kubernetes-mcp-server/pkg/mcp"
)

func Serve(mcpServer *mcp.Server, port, sseBaseUrl string) error {
	mux := http.NewServeMux()
	wrappedMux := RequestMiddleware(mux)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: wrappedMux,
	}

	sseServer := mcpServer.ServeSse(sseBaseUrl, httpServer)
	streamableHttpServer := mcpServer.ServeHTTP(httpServer)
	mux.Handle("/sse", sseServer)
	mux.Handle("/message", sseServer)
	mux.Handle("/mcp", streamableHttpServer)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	klog.V(0).Infof("Streaming and SSE HTTP servers starting on port %s and paths /mcp, /sse, /message", port)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
