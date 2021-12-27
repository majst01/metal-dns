package service

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/majst01/metal-dns/api/v1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const oneYear = time.Hour * 24 * 360

type TokenService struct {
	secret string
	log    *zap.Logger
}

func NewTokenService(l *zap.Logger, secret string) *TokenService {
	return &TokenService{
		secret: secret,
		log:    l,
	}
}
func (t *TokenService) Create(ctx context.Context, req *v1.TokenCreateRequest) (*v1.TokenResponse, error) {
	token, err := newJWTToken("metal-dns", req.Issuer, req.Domains, oneYear, t.secret)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &v1.TokenResponse{Token: token}, nil
}

type dnsClaims struct {
	jwt.RegisteredClaims

	Domains []string `json:"domains,omitempty"`
}

func newJWTToken(subject, issuer string, domains []string, expires time.Duration, secret string) (string, error) {
	now := time.Now().UTC()
	claims := &dnsClaims{
		// see overview of "registered" JWT claims as used by jwt-go here:
		//   https://pkg.go.dev/github.com/golang-jwt/jwt/v4?utm_source=godoc#RegisteredClaims
		// see the semantics of the registered claims here:
		//   https://en.wikipedia.org/wiki/JSON_Web_Token#Standard_fields
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expires)),
			IssuedAt:  jwt.NewNumericDate(now),

			// ID is for your traceability, doesn't have to be UUID:
			ID: uuid.New().String(),

			// put name/title/ID of whoever will be using this JWT here:
			Subject: subject,
			Issuer:  issuer,
		},
		Domains: domains,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	res, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("unable to sign RS256 JWT: %w", err)
	}
	return res, nil
}
