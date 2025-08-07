package http

import (
	"io"
	"net/http"
	"strings"

	"github.com/containers/kubernetes-mcp-server/pkg/config"
)

const (
	oauthAuthorizationServerEndpoint = "/.well-known/oauth-authorization-server"
	oauthProtectedResourceEndpoint   = "/.well-known/oauth-protected-resource"
	openIDConfigurationEndpoint      = "/.well-known/openid-configuration"
)

var WellKnownEndpoints = []string{
	oauthAuthorizationServerEndpoint,
	oauthProtectedResourceEndpoint,
	openIDConfigurationEndpoint,
}

type WellKnown struct {
	authorizationUrl string
}

var _ http.Handler = &WellKnown{}

func WellKnownHandler(staticConfig *config.StaticConfig) http.Handler {
	authorizationUrl := staticConfig.AuthorizationURL
	if authorizationUrl != "" && strings.HasSuffix("authorizationUrl", "/") {
		authorizationUrl = strings.TrimSuffix(authorizationUrl, "/")
	}
	return &WellKnown{authorizationUrl}
}

func (w WellKnown) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if w.authorizationUrl == "" {
		http.Error(writer, "Authorization URL is not configured", http.StatusNotFound)
		return
	}
	req, err := http.NewRequest(request.Method, w.authorizationUrl+request.URL.EscapedPath(), nil)
	if err != nil {
		http.Error(writer, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := http.DefaultClient.Do(req.WithContext(request.Context()))
	if err != nil {
		http.Error(writer, "Failed to perform request: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(writer, "Failed to read response body: "+err.Error(), http.StatusInternalServerError)
		return
	}
	for key, values := range resp.Header {
		for _, value := range values {
			writer.Header().Add(key, value)
		}
	}
	writer.WriteHeader(resp.StatusCode)
	_, _ = writer.Write(body)
}
