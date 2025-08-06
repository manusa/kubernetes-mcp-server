package http

import (
	"encoding/json"
	"net/http"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
	"github.com/containers/kubernetes-mcp-server/pkg/mcp"
)

const (
	oauthProtectedResourceEndpoint = "/.well-known/oauth-protected-resource"
)

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
