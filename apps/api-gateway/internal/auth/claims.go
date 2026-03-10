package auth

import (
	"errors"
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

func ParseAndVerify(tokenString, secret, issuer, audience string) (*Claims, error) {
	mapClaims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, mapClaims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected alg: %s", token.Method.Alg())
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	sub, _ := mapClaims["sub"].(string)
	if sub == "" {
		return nil, errors.New("subject is required")
	}
	iss, _ := mapClaims["iss"].(string)
	if iss != issuer {
		return nil, errors.New("invalid issuer")
	}
	if !validAudience(mapClaims["aud"], audience) {
		return nil, errors.New("invalid audience")
	}
	expUnix, ok := mapClaims["exp"].(float64)
	if !ok {
		return nil, errors.New("exp is required")
	}
	exp := time.Unix(int64(expUnix), 0)
	if exp.Before(time.Now()) {
		return nil, errors.New("token expired")
	}

	scopes := extractScopes(mapClaims["scopes"])
	tenantID, _ := mapClaims["tenant_id"].(string)
	return &Claims{
		Subject:   sub,
		Scopes:    scopes,
		TenantID:  tenantID,
		Issuer:    iss,
		Audience:  audience,
		ExpiresAt: exp,
	}, nil
}

func HasScope(scopes []string, need string) bool {
	for _, s := range scopes {
		if s == need {
			return true
		}
	}
	return false
}

func extractScopes(v any) []string {
	arr, ok := v.([]any)
	if !ok {
		if strArr, ok := v.([]string); ok {
			out := make([]string, 0, len(strArr))
			out = append(out, strArr...)
			return out
		}
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, raw := range arr {
		if s, ok := raw.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func validAudience(v any, expected string) bool {
	s, ok := v.(string)
	if ok {
		return s == expected
	}
	arr, ok := v.([]any)
	if ok {
		for _, raw := range arr {
			if sv, ok := raw.(string); ok && sv == expected {
				return true
			}
		}
	}
	return false
}
