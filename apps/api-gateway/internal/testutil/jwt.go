package testutil

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenParams struct {
	Secret   string
	Issuer   string
	Audience string
	Subject  string
	Scopes   []string
	TenantID string
	TTL      time.Duration
}

func MustIssueToken(t *testing.T, p TokenParams) string {
	t.Helper()
	subject := p.Subject
	if subject == "" {
		subject = "admin-1"
	}
	ttl := p.TTL
	if ttl <= 0 {
		ttl = 1 * time.Hour
	}
	claims := jwt.MapClaims{
		"sub":       subject,
		"iss":       p.Issuer,
		"aud":       p.Audience,
		"exp":       time.Now().Add(ttl).Unix(),
		"scopes":    p.Scopes,
		"tenant_id": p.TenantID,
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(p.Secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}
