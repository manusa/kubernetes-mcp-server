package http

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

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

			token := strings.TrimPrefix(authHeader, "Bearer ")

			err := validateJWTToken(token)
			if err != nil {
				klog.V(1).Infof("Authentication failed - invalid JWT token: %s %s from %s, error: %v", r.Method, r.URL.Path, r.RemoteAddr, err)

				w.Header().Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="Kubernetes MCP Server", audience=%s, error="invalid_token"`, Audience))
				http.Error(w, "Unauthorized: Bearer token required", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type JWTClaims struct {
	Issuer    string   `json:"iss"`
	Audience  []string `json:"aud"`
	ExpiresAt int64    `json:"exp"`
}

// validateJWTToken validates basic JWT claims without signature verification
func validateJWTToken(token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid JWT token format")
	}

	claims, err := parseJWTClaims(parts[1])
	if err != nil {
		return fmt.Errorf("failed to parse JWT claims: %v", err)
	}

	if claims.ExpiresAt > 0 && time.Now().Unix() > claims.ExpiresAt {
		return fmt.Errorf("token expired")
	}

	if !slices.Contains(claims.Audience, Audience) {
		return fmt.Errorf("token audience mismatch: %v", claims.Audience)
	}

	return nil
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
