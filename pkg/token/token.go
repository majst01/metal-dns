package token

import (
	"context"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

type DNSClaims struct {
	jwt.RegisteredClaims

	Domains     []string `json:"domains,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func ParseJWTToken(token string) (*DNSClaims, error) {
	claims := &DNSClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)

	if err != nil {
		return nil, fmt.Errorf("unable to parse token: %q %w", token, err)
	}

	return claims, nil
}

type DNSClaimsKey struct{}

func ClaimsFromContext(ctx context.Context) *DNSClaims {
	return ctx.Value(DNSClaimsKey{}).(*DNSClaims)
}

func ContextWithClaims(ctx context.Context, claims *DNSClaims) context.Context {
	return context.WithValue(ctx, DNSClaimsKey{}, claims)
}
