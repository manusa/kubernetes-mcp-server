package http

import (
	"encoding/base64"
	"encoding/json"
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
		if len(claims.Audience) != 2 {
			t.Errorf("expected 2 audiences, got %d", len(claims.Audience))
		}

		for i, expected := range expectedAudiences {
			if i >= len(claims.Audience) || claims.Audience[i] != expected {
				t.Errorf("expected audience[%d] to be %s, got %s", i, expected, claims.Audience[i])
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
			Audience:  []string{"test-audience"},
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

		err := validateJWTToken(invalidToken)
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
			Audience:  []string{"kubernetes-mcp-server"},
			ExpiresAt: time.Now().Add(-time.Hour).Unix(), // 1 hour ago
		}

		jsonBytes, _ := json.Marshal(expiredClaims)
		payload := base64.URLEncoding.EncodeToString(jsonBytes)
		expiredToken := "header." + payload + ".signature"

		err := validateJWTToken(expiredToken)
		if err == nil {
			t.Error("expected error for expired token, got nil")
		}

		if !strings.Contains(err.Error(), "token expired") {
			t.Errorf("expected expiration error message, got %v", err)
		}
	})

	t.Run("audience mismatch", func(t *testing.T) {
		// Create a token with wrong audience
		wrongAudienceClaims := JWTClaims{
			Issuer:    "test-issuer",
			Audience:  []string{"wrong-audience", "another-wrong-audience"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		jsonBytes, _ := json.Marshal(wrongAudienceClaims)
		payload := base64.URLEncoding.EncodeToString(jsonBytes)
		wrongAudienceToken := "header." + payload + ".signature"

		err := validateJWTToken(wrongAudienceToken)
		if err == nil {
			t.Error("expected error for audience mismatch, got nil")
		}

		if !strings.Contains(err.Error(), "token audience mismatch") {
			t.Errorf("expected audience mismatch error message, got %v", err)
		}
	})

	t.Run("multiple audiences with correct one", func(t *testing.T) {
		// Create a token with multiple audiences including the correct one
		multiAudienceClaims := JWTClaims{
			Issuer:    "test-issuer",
			Audience:  []string{"other-audience", "kubernetes-mcp-server", "third-audience"},
			ExpiresAt: time.Now().Add(time.Hour).Unix(),
		}

		jsonBytes, _ := json.Marshal(multiAudienceClaims)
		payload := base64.URLEncoding.EncodeToString(jsonBytes)
		multiAudienceToken := "header." + payload + ".signature"

		err := validateJWTToken(multiAudienceToken)
		if err != nil {
			t.Errorf("expected no error for token with correct audience among multiple, got %v", err)
		}
	})
}
