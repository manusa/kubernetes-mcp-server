package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"k8s.io/klog/v2"

	"github.com/containers/kubernetes-mcp-server/pkg/mcp"
)

const (
	Audience = "kubernetes-mcp-server"
)

// AuthorizationMiddleware validates the OAuth flow using Kubernetes TokenReview API
func AuthorizationMiddleware(requireOAuth bool, serverURL string, oidcProvider *oidc.Provider, mcpServer *mcp.Server) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == healthEndpoint || r.URL.Path == oauthProtectedResourceEndpoint || r.URL.Path == oauthAuthorizationServerEndpoint {
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
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s", error="missing_token"`, audience))
				} else {
					w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s"", resource_metadata="%s%s", error="missing_token"`, audience, serverURL, oauthProtectedResourceEndpoint))
				}
				http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			// Validate the token offline for simple sanity check
			// Because missing expected audience and expired tokens must be
			// rejected already.
			claims, err := ParseJWTClaims(token)
			if err == nil && claims != nil {
				err = claims.Validate(r.Context(), audience, oidcProvider)
			}
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

			// Scopes are likely to be used for authorization.
			scopes := claims.GetScopes()
			klog.V(2).Infof("JWT token validated - Scopes: %v", scopes)
			r = r.WithContext(context.WithValue(r.Context(), mcp.TokenScopesContextKey, scopes))

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
				klog.V(1).Infof("Authentication failed - API Server token validation error: %s %s from %s, error: %v", r.Method, r.URL.Path, r.RemoteAddr, err)

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

var allSignatureAlgorithms = []jose.SignatureAlgorithm{
	jose.EdDSA,
	jose.HS256,
	jose.HS384,
	jose.HS512,
	jose.RS256,
	jose.RS384,
	jose.RS512,
	jose.ES256,
	jose.ES384,
	jose.ES512,
	jose.PS256,
	jose.PS384,
	jose.PS512,
}

type JWTClaims struct {
	jwt.Claims
	Token string `json:"-"`
	Scope string `json:"scope,omitempty"`
}

func (c *JWTClaims) GetScopes() []string {
	if c.Scope == "" {
		return nil
	}
	return strings.Fields(c.Scope)
}

// Validate Checks if the JWT claims are valid and if the audience matches the expected one.
func (c *JWTClaims) Validate(ctx context.Context, audience string, provider *oidc.Provider) error {
	if err := c.Claims.Validate(jwt.Expected{AnyAudience: jwt.Audience{audience}}); err != nil {
		return fmt.Errorf("JWT token validation error: %v", err)
	}
	if provider != nil {
		verifier := provider.Verifier(&oidc.Config{
			ClientID: audience,
		})

		_, err := verifier.Verify(ctx, c.Token)
		if err != nil {
			return fmt.Errorf("OIDC token validation error: %v", err)
		}
	}
	return nil
}

func ParseJWTClaims(token string) (*JWTClaims, error) {
	tkn, err := jwt.ParseSigned(token, allSignatureAlgorithms)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JWT token: %w", err)
	}
	claims := &JWTClaims{}
	err = tkn.UnsafeClaimsWithoutVerification(claims)
	claims.Token = token
	return claims, err
}
