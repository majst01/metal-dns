package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	v1 "github.com/majst01/metal-dns/api/v1"
	"go.uber.org/zap"
)

// Client defines the client API
type Client interface {
	Domain() v1.DomainServiceClient
	Record() v1.RecordServiceClient
	Token() v1.TokenServiceClient
	Close() error
}

// GRPCClient is a Client implementation with grpc transport.
type GRPCClient struct {
	conn *grpc.ClientConn
	log  *zap.Logger
}

// NewClient creates a new client for the services for the given address, with the certificate and hmac.
func NewClient(ctx context.Context, hostname string, port int, certFile string, keyFile string, caFile string, token string, logger *zap.Logger) (Client, error) {

	address := fmt.Sprintf("%s:%d", hostname, port)

	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system credentials: %w", err)
	}

	if caFile != "" {
		ca, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("could not read ca certificate: %w", err)
		}

		ok := certPool.AppendCertsFromPEM(ca)
		if !ok {
			return nil, fmt.Errorf("failed to append ca certs: %s", caFile)
		}
	}

	clientCertificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("could not load client key pair: %w", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   hostname,
		Certificates: []tls.Certificate{clientCertificate},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
	})

	client := GRPCClient{
		log: logger,
	}

	opts := []grpc.DialOption{
		// oauth.NewOauthAccess requires the configuration of transport
		// credentials.
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(tokenAuth{
			token: token,
		}),

		// grpc.WithInsecure(),
		grpc.WithBlock(),
	}
	// Set up a connection to the server.
	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return nil, err
	}
	client.conn = conn

	return client, nil
}

type tokenAuth struct {
	token string
}

func (t tokenAuth) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": "Bearer " + t.token,
	}, nil
}

func (tokenAuth) RequireTransportSecurity() bool {
	return true
}

// Close the underlying connection
func (c GRPCClient) Close() error {
	return c.conn.Close()
}

// Domain is the root accessor for domain related functions
func (c GRPCClient) Domain() v1.DomainServiceClient {
	return v1.NewDomainServiceClient(c.conn)
}

// Record is the root accessor for domain record related functions
func (c GRPCClient) Record() v1.RecordServiceClient {
	return v1.NewRecordServiceClient(c.conn)
}

// Token is the root accessor for domain record related functions
func (c GRPCClient) Token() v1.TokenServiceClient {
	return v1.NewTokenServiceClient(c.conn)
}
