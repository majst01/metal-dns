package service

import (
	"context"
	"fmt"
	"time"

	connect "github.com/bufbuild/connect-go"

	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/token"
	"go.uber.org/zap"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

const oneYear = time.Hour * 24 * 360

type TokenService struct {
	secret string
	log    *zap.SugaredLogger
}

func NewTokenService(l *zap.SugaredLogger, secret string) *TokenService {
	return &TokenService{
		secret: secret,
		log:    l.Named("token"),
	}
}
func (t *TokenService) Create(ctx context.Context, rq *connect.Request[v1.TokenServiceCreateRequest]) (*connect.Response[v1.TokenServiceCreateResponse], error) {
	t.log.Debugw("create", "req", rq)
	req := rq.Msg
	exp := oneYear
	if req.Expires != nil {
		exp = req.Expires.AsDuration()
	}
	token, err := newJWTToken("metal-dns", req.Issuer, req.Domains, req.Permissions, exp, t.secret)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.TokenServiceCreateResponse{Token: token}), nil
}

func newJWTToken(subject, issuer string, domains, permissions []string, expires time.Duration, secret string) (string, error) {
	now := time.Now().UTC()
	claims := &token.DNSClaims{
		// see overview of "registered" JWT claims as used by jwt-go here:
		//   https://pkg.go.dev/github.com/golang-jwt/jwt/v4?utm_source=godoc#RegisteredClaims
		// see the semantics of the registered claims here:
		//   https://en.wikipedia.org/wiki/JSON_Web_Token#Standard_fields
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expires)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),

			// ID is for your traceability, doesn't have to be UUID:
			ID: uuid.New().String(),

			// put name/title/ID of whoever will be using this JWT here:
			Subject: subject,
			Issuer:  issuer,
		},
		Domains:     domains,
		Permissions: permissions,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	res, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("unable to sign RS256 JWT: %w", err)
	}
	return res, nil
}
