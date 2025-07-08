package http

import (
	"fmt"
	"net/http"
	"strings"

	"k8s.io/klog/v2"
)

const (
	Audience = "kubernetes-mcp-server"
)

// AuthorizationMiddleware validates the OAuth flow
func AuthorizationMiddleware(requireOAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" {
				next.ServeHTTP(w, r)
				return
			}
			if !requireOAuth {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				klog.V(1).Infof("Authentication failed - missing or invalid bearer token: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience=%s, error="invalid_token"`, Audience))
				http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
