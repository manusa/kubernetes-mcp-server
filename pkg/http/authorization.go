package http

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"k8s.io/klog/v2"

	"github.com/manusa/kubernetes-mcp-server/pkg/mcp"
)

const (
	Audience = "kubernetes-mcp-server"
)

// AuthorizationMiddleware validates the OAuth flow using Kubernetes TokenReview API
func AuthorizationMiddleware(requireOAuth bool, serverURL string, oidcProvider *oidc.Provider, mcpServer *mcp.Server) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/healthz" || r.URL.Path == "/.well-known/oauth-protected-resource" {
				next.ServeHTTP(w, r)
				return
			}
			if !requireOAuth {
				next.ServeHTTP(w, r)
				return
			}

			audience := Audience
			if serverURL != "" {
				audience = serverURL
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				klog.V(1).Infof("Authentication failed - missing or invalid bearer token: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

				if serverURL == "" {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s", error="invalid_token"`, audience))
				} else {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s"", resource_metadata="%s%s", error="invalid_token"`, audience, serverURL, oauthProtectedResourceEndpoint))
				}
				http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate the token offline for simple sanity check
			// Because missing expected audience and expired tokens must be
			// rejected already.
			claims, err := validateJWTToken(token, audience)
			if err != nil {
				klog.V(1).Infof("Authentication failed - JWT validation error: %s %s from %s, error: %v", r.Method, r.URL.Path, r.RemoteAddr, err)

				if serverURL == "" {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s", error="invalid_token"`, audience))
				} else {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s"", resource_metadata="%s%s", error="invalid_token"`, audience, serverURL, oauthProtectedResourceEndpoint))
				}
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}
			
			if oidcProvider != nil {
				// If OIDC Provider is configured, this token must be validated against it.
				if err := validateTokenWithOIDC(r.Context(), oidcProvider, token, audience); err != nil {
					klog.V(1).Infof("Authentication failed - OIDC token validation error: %s %s from %s, error: %v", r.Method, r.URL.Path, r.RemoteAddr, err)

					if serverURL == "" {
						w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s", error="invalid_token"`, audience))
					} else {
						w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s"", resource_metadata="%s%s", error="invalid_token"`, audience, serverURL, oauthProtectedResourceEndpoint))
					}
					http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
					return
				}
			}

			// Scopes are likely to be used for authorization.
			scopes := claims.GetScopes()
			klog.V(2).Infof("JWT token validated - Scopes: %v", scopes)

			// Now, there are a couple of options:
			// 1. If there is no authorization url configured for this MCP Server,
			// that means this token will be used against the Kubernetes API Server.
			// So that we need to validate the token using Kubernetes TokenReview API beforehand.
			// 2. If there is an authorization url configured for this MCP Server,
			// that means up to this point, the token is validated against the OIDC Provider already.
			// 2. a. If this is the only token in the headers, this validated token
			// is supposed to be used against the Kubernetes API Server as well. Therefore,
			// TokenReview request must succeed.
			// 2. b. If this is not the only token in the headers, the token in here is used
			// only for authentication and authorization. Therefore, we need to send TokenReview request
			// with the other token in the headers (TODO: still need to validate aud and exp of this token separately).
			_, _, err = mcpServer.VerifyTokenAPIServer(r.Context(), token, audience)
			if err != nil {
				klog.V(1).Infof("Authentication failed - token validation error: %s %s from %s, error: %v", r.Method, r.URL.Path, r.RemoteAddr, err)

				if serverURL == "" {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s", error="invalid_token"`, audience))
				} else {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s"", resource_metadata="%s%s", error="invalid_token"`, audience, serverURL, oauthProtectedResourceEndpoint))
				}
				http.Error(w, "Unauthorized: Invalid token", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type JWTClaims struct {
	Issuer    string `json:"iss"`
	Audience  any    `json:"aud"`
	ExpiresAt int64  `json:"exp"`
	Scope     string `json:"scope,omitempty"`
}

func (c *JWTClaims) GetScopes() []string {
	if c.Scope == "" {
		return nil
	}
	return strings.Fields(c.Scope)
}

func (c *JWTClaims) ContainsAudience(audience string) bool {
	switch aud := c.Audience.(type) {
	case string:
		return aud == audience
	case []interface{}:
		for _, a := range aud {
			if str, ok := a.(string); ok && str == audience {
				return true
			}
		}
	case []string:
		for _, a := range aud {
			if a == audience {
				return true
			}
		}
	}
	return false
}

// validateJWTToken validates basic JWT claims without signature verification and returns the claims
func validateJWTToken(token, audience string) (*JWTClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT token format")
	}

	claims, err := parseJWTClaims(parts[1])
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT claims: %v", err)
	}

	if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	if !claims.ContainsAudience(audience) {
		return nil, fmt.Errorf("token audience mismatch: %v", claims.Audience)
	}

	return claims, nil
}

func parseJWTClaims(payload string) (*JWTClaims, error) {
	// Add padding if needed
	if len(payload)%4 != 0 {
		payload += strings.Repeat("=", 4-len(payload)%4)
	}

	decoded, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JWT payload: %v", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT claims: %v", err)
	}

	return &claims, nil
}

func validateTokenWithOIDC(ctx context.Context, provider *oidc.Provider, token, audience string) error {
	verifier := provider.Verifier(&oidc.Config{
		ClientID: audience,
	})

	_, err := verifier.Verify(ctx, token)
	if err != nil {
		return fmt.Errorf("JWT token verification failed: %v", err)
	}

	return nil
}
