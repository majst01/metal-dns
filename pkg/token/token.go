package token

import "github.com/golang-jwt/jwt/v4"

type DNSClaims struct {
	jwt.RegisteredClaims

	Domains     []string `json:"domains,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func ParseJWTToken(token string) (*DNSClaims, error) {
	claims := &DNSClaims{}
	_, _, err := new(jwt.Parser).ParseUnverified(string(token), claims)

	if err != nil {
		return nil, err
	}

	return claims, nil
}
