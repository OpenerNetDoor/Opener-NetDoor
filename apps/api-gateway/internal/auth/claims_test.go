package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestParseAndVerifyHappyPath(t *testing.T) {
	secret := "very-secure-secret"
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":       "admin-1",
		"iss":       "iss",
		"aud":       "aud",
		"exp":       time.Now().Add(1 * time.Hour).Unix(),
		"scopes":    []string{"admin:read"},
		"tenant_id": "tenant-1",
	})
	signed, err := tok.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	claims, err := ParseAndVerify(signed, secret, "iss", "aud")
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	if claims.Subject != "admin-1" {
		t.Fatalf("unexpected subject %q", claims.Subject)
	}
	if claims.TenantID != "tenant-1" {
		t.Fatalf("unexpected tenant_id %q", claims.TenantID)
	}
	if !HasScope(claims.Scopes, "admin:read") {
		t.Fatalf("expected admin:read scope")
	}
}

func TestParseAndVerifyExpired(t *testing.T) {
	secret := "very-secure-secret"
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":    "admin-1",
		"iss":    "iss",
		"aud":    "aud",
		"exp":    time.Now().Add(-1 * time.Hour).Unix(),
		"scopes": []string{"admin:read"},
	})
	signed, _ := tok.SignedString([]byte(secret))
	if _, err := ParseAndVerify(signed, secret, "iss", "aud"); err == nil {
		t.Fatal("expected verify error for expired token")
	}
}
