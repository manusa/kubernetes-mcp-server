package http

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseJWTClaims(t *testing.T) {
	t.Run("valid JWT payload", func(t *testing.T) {
		// Sample payload from a valid JWT
		payload := "eyJhdWQiOlsiaHR0cHM6Ly9rdWJlcm5ldGVzLmRlZmF1bHQuc3ZjLmNsdXN0ZXIubG9jYWwiLCJrdWJlcm5ldGVzLW1jcC1zZXJ2ZXIiXSwiZXhwIjoxNzUxOTYzOTQ4LCJpYXQiOjE3NTE5NjAzNDgsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2Yy5jbHVzdGVyLmxvY2FsIiwianRpIjoiOTkyMjJkNTYtMzQwZS00ZWI2LTg1ODgtMjYxNDExZjM1ZDI2Iiwia3ViZXJuZXRlcy5pbyI6eyJuYW1lc3BhY2UiOiJkZWZhdWx0Iiwic2VydmljZWFjY291bnQiOnsibmFtZSI6ImRlZmF1bHQiLCJ1aWQiOiJlYWNiNmFkMi04MGI3LTQxNzktODQzZC05MmViMWU2YmJiYTYifX0sIm5iZiI6MTc1MTk2MDM0OCwic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50OmRlZmF1bHQ6ZGVmYXVsdCJ9"

		claims, err := parseJWTClaims(payload)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if claims == nil {
			t.Fatal("expected claims, got nil")
		}

		if claims.Issuer != "https://kubernetes.default.svc.cluster.local" {
			t.Errorf("expected issuer 'https://kubernetes.default.svc.cluster.local', got %s", claims.Issuer)
		}

		expectedAudiences := []string{"https://kubernetes.default.svc.cluster.local", "kubernetes-mcp-server"}
		for _, expected := range expectedAudiences {
			if !claims.ContainsAudience(expected) {
				t.Errorf("expected audience to contain %s", expected)
			}
		}

		if claims.ExpiresAt != 1751963948 {
			t.Errorf("expected exp 1751963948, got %d", claims.ExpiresAt)
		}
	})

	t.Run("payload needs padding", func(t *testing.T) {
		// Create a payload that needs padding
		testClaims := JWTClaims{
			Issuer:    "test-issuer",
			Audience:  "test-audience",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		jsonBytes, _ := json.Marshal(testClaims)
		// Create a payload without proper padding
		encodedWithoutPadding := strings.TrimRight(base64.URLEncoding.EncodeToString(jsonBytes), "=")

		claims, err := parseJWTClaims(encodedWithoutPadding)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if claims.Issuer != "test-issuer" {
			t.Errorf("expected issuer 'test-issuer', got %s", claims.Issuer)
		}
	})

	t.Run("invalid base64 payload", func(t *testing.T) {
		invalidPayload := "invalid-base64!!!"

		_, err := parseJWTClaims(invalidPayload)
		if err == nil {
			t.Error("expected error for invalid base64, got nil")
		}

		if !strings.Contains(err.Error(), "failed to decode JWT payload") {
			t.Errorf("expected decode error message, got %v", err)
		}
	})

	t.Run("invalid JSON payload", func(t *testing.T) {
		// Valid base64 but invalid JSON
		invalidJSON := base64.URLEncoding.EncodeToString([]byte("{invalid-json"))

		_, err := parseJWTClaims(invalidJSON)
		if err == nil {
			t.Error("expected error for invalid JSON, got nil")
		}

		if !strings.Contains(err.Error(), "failed to unmarshal JWT claims") {
			t.Errorf("expected unmarshal error message, got %v", err)
		}
	})
}

func TestValidateJWTToken(t *testing.T) {
	t.Run("invalid token format - not enough parts", func(t *testing.T) {
		invalidToken := "header.payload"

		_, err := validateJWTToken(invalidToken, "test")
		if err == nil {
			t.Error("expected error for invalid token format, got nil")
		}

		if !strings.Contains(err.Error(), "invalid JWT token format") {
			t.Errorf("expected format error message, got %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		// Create an expired token
		expiredClaims := JWTClaims{
			Issuer:    "test-issuer",
			Audience:  "kubernetes-mcp-server",
			ExpiresAt: time.Now().Add(-time.Hour).Unix(),
		}

		jsonBytes, _ := json.Marshal(expiredClaims)
		payload := base64.URLEncoding.EncodeToString(jsonBytes)
		expiredToken := "header." + payload + ".signature"

		_, err := validateJWTToken(expiredToken, "kubernetes-mcp-server")
		if err == nil {
			t.Error("expected error for expired token, got nil")
		}

		if !strings.Contains(err.Error(), "token expired") {
			t.Errorf("expected expiration error message, got %v", err)
		}
	})

	t.Run("multiple audiences with correct one", func(t *testing.T) {
		// Create a token with multiple audiences including the correct one
		multiAudClaims := JWTClaims{
			Issuer:    "test-issuer",
			Audience:  []string{"other-audience", "kubernetes-mcp-server", "another-audience"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
			Scope:     "read write admin",
		}

		jsonBytes, _ := json.Marshal(multiAudClaims)
		payload := base64.URLEncoding.EncodeToString(jsonBytes)
		multiAudToken := "header." + payload + ".signature"

		claims, err := validateJWTToken(multiAudToken, "kubernetes-mcp-server")
		if err != nil {
			t.Errorf("expected no error for token with multiple audiences, got %v", err)
		}
		if claims == nil {
			t.Error("expected claims to be returned, got nil")
		}
		if claims.Issuer != "test-issuer" {
			t.Errorf("expected issuer 'test-issuer', got %s", claims.Issuer)
		}

		// Test scope parsing
		scopes := claims.GetScopes()
		expectedScopes := []string{"read", "write", "admin"}
		if len(scopes) != 3 {
			t.Errorf("expected 3 scopes, got %d", len(scopes))
		}
		for i, expectedScope := range expectedScopes {
			if i >= len(scopes) || scopes[i] != expectedScope {
				t.Errorf("expected scope[%d] to be '%s', got '%s'", i, expectedScope, scopes[i])
			}
		}
	})

	t.Run("audience mismatch", func(t *testing.T) {
		// Create a token with wrong audience
		wrongAudClaims := JWTClaims{
			Issuer:    "test-issuer",
			Audience:  "wrong-audience",
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		jsonBytes, _ := json.Marshal(wrongAudClaims)
		payload := base64.URLEncoding.EncodeToString(jsonBytes)
		wrongAudToken := "header." + payload + ".signature"

		_, err := validateJWTToken(wrongAudToken, "audience")
		if err == nil {
			t.Error("expected error for token with wrong audience, got nil")
		}

		if !strings.Contains(err.Error(), "audience mismatch") {
			t.Errorf("expected audience mismatch error, got %v", err)
		}
	})
}

func TestAuthorizationMiddleware(t *testing.T) {
	// Create a mock handler
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	t.Run("OAuth disabled - passes through", func(t *testing.T) {
		handlerCalled = false

		// Create middleware with OAuth disabled
		middleware := AuthorizationMiddleware(false, "", nil)
		wrappedHandler := middleware(handler)

		// Create request without authorization header
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("expected handler to be called when OAuth is disabled")
		}
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("healthz endpoint - passes through", func(t *testing.T) {
		handlerCalled = false

		// Create middleware with OAuth enabled
		middleware := AuthorizationMiddleware(true, "", nil)
		wrappedHandler := middleware(handler)

		// Create request to healthz endpoint
		req := httptest.NewRequest("GET", "/healthz", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if !handlerCalled {
			t.Error("expected handler to be called for healthz endpoint")
		}
		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}
	})

	t.Run("OAuth enabled - missing token", func(t *testing.T) {
		handlerCalled = false

		// Create middleware with OAuth enabled
		middleware := AuthorizationMiddleware(true, "", nil)
		wrappedHandler := middleware(handler)

		// Create request without authorization header
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("expected handler NOT to be called when token is missing")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Bearer token required") {
			t.Errorf("expected bearer token error message, got %s", w.Body.String())
		}
	})

	t.Run("OAuth enabled - invalid token format", func(t *testing.T) {
		handlerCalled = false

		// Create middleware with OAuth enabled
		middleware := AuthorizationMiddleware(true, "", nil)
		wrappedHandler := middleware(handler)

		// Create request with invalid bearer token
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(w, req)

		if handlerCalled {
			t.Error("expected handler NOT to be called when token is invalid")
		}
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected status 401, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "Invalid token") {
			t.Errorf("expected invalid token error message, got %s", w.Body.String())
		}
	})
}

func TestJWTClaimsGetScopes(t *testing.T) {
	t.Run("single scope", func(t *testing.T) {
		claims := &JWTClaims{Scope: "read"}
		scopes := claims.GetScopes()
		expected := []string{"read"}

		if len(scopes) != 1 {
			t.Errorf("expected 1 scope, got %d", len(scopes))
		}
		if scopes[0] != expected[0] {
			t.Errorf("expected scope 'read', got '%s'", scopes[0])
		}
	})

	t.Run("multiple scopes", func(t *testing.T) {
		claims := &JWTClaims{Scope: "read write admin"}
		scopes := claims.GetScopes()
		expected := []string{"read", "write", "admin"}

		if len(scopes) != 3 {
			t.Errorf("expected 3 scopes, got %d", len(scopes))
		}

		for i, expectedScope := range expected {
			if i >= len(scopes) || scopes[i] != expectedScope {
				t.Errorf("expected scope[%d] to be '%s', got '%s'", i, expectedScope, scopes[i])
			}
		}
	})

	t.Run("scopes with extra whitespace", func(t *testing.T) {
		claims := &JWTClaims{Scope: "  read   write   admin  "}
		scopes := claims.GetScopes()
		expected := []string{"read", "write", "admin"}

		if len(scopes) != 3 {
			t.Errorf("expected 3 scopes, got %d", len(scopes))
		}

		for i, expectedScope := range expected {
			if i >= len(scopes) || scopes[i] != expectedScope {
				t.Errorf("expected scope[%d] to be '%s', got '%s'", i, expectedScope, scopes[i])
			}
		}
	})
}

func TestJWTClaimsContainsAudience(t *testing.T) {
	t.Run("single string audience", func(t *testing.T) {
		claims := &JWTClaims{Audience: "test-audience"}

		if !claims.ContainsAudience("test-audience") {
			t.Error("expected ContainsAudience to return true for matching audience")
		}

		if claims.ContainsAudience("other-audience") {
			t.Error("expected ContainsAudience to return false for non-matching audience")
		}
	})

	t.Run("array audience", func(t *testing.T) {
		claims := &JWTClaims{Audience: []string{"aud1", "aud2", "aud3"}}

		testCases := []struct {
			audience string
			expected bool
		}{
			{"aud1", true},
			{"aud2", true},
			{"aud3", true},
			{"aud4", false},
			{"", false},
		}

		for _, tc := range testCases {
			if claims.ContainsAudience(tc.audience) != tc.expected {
				t.Errorf("expected ContainsAudience(%s) to return %v", tc.audience, tc.expected)
			}
		}
	})
}
