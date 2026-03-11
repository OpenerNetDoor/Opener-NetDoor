package session

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	Subject   string
	Scopes    []string
	TenantID  string
	Issuer    string
	Audience  string
	ExpiresAt time.Time
}

func Issue(secret string, issuer string, audience string, ttl time.Duration, subject string, tenantID string, scopes []string) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	claims := jwt.MapClaims{
		"sub":       subject,
		"iss":       issuer,
		"aud":       audience,
		"iat":       now.Unix(),
		"exp":       expiresAt.Unix(),
		"tenant_id": tenantID,
		"scopes":    scopes,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	raw, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign session token: %w", err)
	}
	return raw, expiresAt, nil
}
