package http

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/mcp"
)

const (
	oauthAuthorizationServerEndpoint = "/.well-known/oauth-authorization-server"
	oauthProtectedResourceEndpoint   = "/.well-known/oauth-protected-resource"
)

func OAuthAuthorizationServerHandler(staticConfig *config.StaticConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if staticConfig.AuthorizationURL == "" {
			http.Error(w, "Authorization URL is not configured", http.StatusNotFound)
			return
		}
		req, err := http.NewRequest(r.Method, staticConfig.AuthorizationURL+oauthAuthorizationServerEndpoint, nil)
		if err != nil {
			http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
			return
		}
		resp, err := http.DefaultClient.Do(req.WithContext(r.Context()))
		if err != nil {
			http.Error(w, "Failed to perform request: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write(body)
	}
}

func OAuthProtectedResourceHandler(mcpServer *mcp.Server, staticConfig *config.StaticConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var authServers []string
		if staticConfig.AuthorizationURL != "" {
			authServers = []string{staticConfig.AuthorizationURL}
		} else {
			// Fallback to Kubernetes API server host if authorization_server is not configured
			if apiServerHost := mcpServer.GetKubernetesAPIServerHost(); apiServerHost != "" {
				authServers = []string{apiServerHost}
			}
		}

		response := map[string]interface{}{
			"authorization_servers":    authServers,
			"authorization_server":     authServers[0],
			"scopes_supported":         mcpServer.GetEnabledTools(),
			"bearer_methods_supported": []string{"header"},
		}

		if staticConfig.ServerURL != "" {
			response["resource"] = staticConfig.ServerURL
		}

		if staticConfig.JwksURL != "" {
			response["jwks_uri"] = staticConfig.JwksURL
		}

		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
