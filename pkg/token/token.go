package token

import (
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

type DNSClaims struct {
	jwt.RegisteredClaims

	Domains     []string `json:"domains,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type DNSClaimsKey struct{}

func ParseJWTToken(token string) (*DNSClaims, error) {
	claims := &DNSClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(token, claims)

	if err != nil {
		return nil, fmt.Errorf("unable to parse token: %q %w", token, err)
	}

	return claims, nil
}
