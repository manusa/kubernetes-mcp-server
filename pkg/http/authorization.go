package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	authenticationapiv1 "k8s.io/api/authentication/v1"
	"k8s.io/klog/v2"
	"k8s.io/utils/strings/slices"

	"github.com/containers/kubernetes-mcp-server/pkg/mcp"
)

const (
	Audience = "mcp-server"
)

type KubernetesApiTokenVerifier interface {
	// KubernetesApiVerifyToken TODO: clarify proper implementation
	KubernetesApiVerifyToken(ctx context.Context, token, audience string) (*authenticationapiv1.UserInfo, []string, error)
}

// AuthorizationMiddleware validates the OAuth flow for protected resources.
//
// The flow is skipped for unprotected resources, such as health checks and well-known endpoints.
//
//	There are several auth scenarios
//
//	 1. requireOAuth is false:
//
//	    - The OAuth flow is skipped, and the server is effectively unprotected.
//	    - The request is passed to the next handler without any validation.
//
//	    see TestAuthorizationRequireOAuthFalse
//
//	 2. requireOAuth is set to true, server is protected:
//
//	    2.1. Raw Token Validation (oidcProvider is nil):
//	         - The token is validated offline for basic sanity checks (expiration).
//	         - If audience is set, the token is validated against the audience.
//	         - The token is then used against the Kubernetes API Server for TokenReview.
//
//	    2.2. OIDC Provider Validation (oidcProvider is not nil):
//	         - The token is validated offline for basic sanity checks (audience and expiration).
//	         - The token is then validated against the OIDC Provider.
//	         - The token is then used against the Kubernetes API Server for TokenReview.
//
//	    2.3. OIDC Token Exchange (oidcProvider is not nil and xxx):
func AuthorizationMiddleware(requireOAuth bool, audience string, oidcProvider *oidc.Provider, verifier KubernetesApiTokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == healthEndpoint || slices.Contains(WellKnownEndpoints, r.URL.EscapedPath()) {
				next.ServeHTTP(w, r)
				return
			}
			if !requireOAuth {
				next.ServeHTTP(w, r)
				return
			}

			wwwAuthenticateHeader := "Bearer realm=\"Kubernetes MCP Server\""
			if audience != "" {
				wwwAuthenticateHeader += fmt.Sprintf(`, audience="%s"`, audience)
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				klog.V(1).Infof("Authentication failed - missing or invalid bearer token: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)

				w.Header().Set("WWW-Authenticate", wwwAuthenticateHeader+", error=\"missing_token\"")
				http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
				return
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := ParseJWTClaims(token)
			if err == nil && claims != nil {
				err = claims.ValidateOffline(audience)
			}
			if err == nil && claims != nil {
				err = claims.ValidateWithProvider(r.Context(), audience, oidcProvider)
			}
			if err != nil {
				klog.V(1).Infof("Authentication failed - JWT validation error: %s %s from %s, error: %v", r.Method, r.URL.Path, r.RemoteAddr, err)

				w.Header().Set("WWW-Authenticate", wwwAuthenticateHeader+", error=\"invalid_token\"")
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
			_, _, err = verifier.KubernetesApiVerifyToken(r.Context(), token, audience)
			if err != nil {
				klog.V(1).Infof("Authentication failed - API Server token validation error: %s %s from %s, error: %v", r.Method, r.URL.Path, r.RemoteAddr, err)

				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience="%s", error="invalid_token"`, audience))
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

// ValidateOffline Checks if the JWT claims are valid and if the audience matches the expected one.
func (c *JWTClaims) ValidateOffline(audience string) error {
	expected := jwt.Expected{}
	if audience != "" {
		expected.AnyAudience = jwt.Audience{audience}
	}
	if err := c.Validate(expected); err != nil {
		return fmt.Errorf("JWT token validation error: %v", err)
	}
	return nil
}

// ValidateWithProvider validates the JWT claims against the OIDC provider.
func (c *JWTClaims) ValidateWithProvider(ctx context.Context, audience string, provider *oidc.Provider) error {
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
