package client

import (
	"context"

	"go.uber.org/zap"

	compress "github.com/klauspost/connect-compress"
	"github.com/majst01/metal-dns/api/v1/apiv1connect"
)

// Client defines the client API
type Client interface {
	Domain() apiv1connect.DomainServiceClient
	Record() apiv1connect.RecordServiceClient
	Token() apiv1connect.TokenServiceClient
}

type api struct {
	log                 *zap.SugaredLogger
	domainServiceClient apiv1connect.DomainServiceClient
	recordServiceClient apiv1connect.RecordServiceClient
	tokenServiceClient  apiv1connect.TokenServiceClient
}

func New(ctx context.Context, config DialConfig) Client {
	return &api{
		log: config.Log,
		domainServiceClient: apiv1connect.NewDomainServiceClient(
			config.HttpClient(),
			config.BaseURL,
			compress.WithAll(compress.LevelBalanced),
		),
		recordServiceClient: apiv1connect.NewRecordServiceClient(
			config.HttpClient(),
			config.BaseURL,
			compress.WithAll(compress.LevelBalanced),
		),
		tokenServiceClient: apiv1connect.NewTokenServiceClient(
			config.HttpClient(),
			config.BaseURL,
			compress.WithAll(compress.LevelBalanced),
		),
	}
}

// Domain is the root accessor for domain related functions
func (a *api) Domain() apiv1connect.DomainServiceClient {
	return a.domainServiceClient
}

// Record is the root accessor for domain record related functions
func (a *api) Record() apiv1connect.RecordServiceClient {
	return a.recordServiceClient
}

// Token is the root accessor for domain record related functions
func (a *api) Token() apiv1connect.TokenServiceClient {
	return a.tokenServiceClient
}
