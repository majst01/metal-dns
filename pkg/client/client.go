package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"

	v1 "github.com/majst01/metal-dns/api/v1"
)

// Client defines the client API
type Client interface {
	Domain() v1.DomainServiceClient
	Record() v1.RecordServiceClient
	Token() v1.TokenServiceClient
	Close() error
}

type DialConfig struct {
	// Address if the metal-dns grpc api endpoint in th form of hostname/ip:port
	// defaults to dns.metal-stack.io:50001 if omitted
	Address *string

	// Token which is used to authenticate against metal-dns api
	// is in the form of a jwt token
	Token string

	Credentials *Credentials

	ByteCredentials *ByteCredentials
}

// Credentials specify the TLS Certificate based authentication for the grpc connection
// If you provide credentials, provide either these or byte credentials but not both.
type Credentials struct {
	ServerName string
	Certfile   string
	Keyfile    string
	CAFile     string
}

// Credentials specify the TLS Certificate based authentication for the grpc connection
// without having to use certificate files.
// If you provide credentials, provide either these or file path credentials but not both.
type ByteCredentials struct {
	ServerName string
	Cert       []byte
	Key        []byte
	CA         []byte
}

// dnsClient is a Client implementation with grpc transport.
type dnsClient struct {
	conn *grpc.ClientConn
}

// NewClient creates a new client for the services for the given address, with the certificate and hmac.
func NewClient(ctx context.Context, config DialConfig) (Client, error) {
	if config.Credentials != nil && config.ByteCredentials != nil {
		return nil, status.Errorf(codes.InvalidArgument,
			"if you provide credentials, provide either file or byte credentials but not both")
	}

	address := "localhost:50051"
	if config.Address != nil {
		address = *config.Address
	}

	opts := []grpc.DialOption{
		grpc.WithUserAgent("metal-dns-go"),
		grpc.WithPerRPCCredentials(tokenAuth{
			token: config.Token,
		}),
	}

	if config.Credentials != nil {
		creds, err := config.Credentials.getTransportCredentials()
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else if config.ByteCredentials != nil {
		creds, err := config.ByteCredentials.getTransportCredentials()
		if err != nil {
			return nil, err
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		//nolint
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	}

	// Set up a connection to the server.
	conn, err := grpc.DialContext(ctx, address, opts...)
	if err != nil {
		return nil, err
	}

	return dnsClient{
		conn: conn,
	}, nil
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
func (c dnsClient) Close() error {
	return c.conn.Close()
}

// Domain is the root accessor for domain related functions
func (c dnsClient) Domain() v1.DomainServiceClient {
	return v1.NewDomainServiceClient(c.conn)
}

// Record is the root accessor for domain record related functions
func (c dnsClient) Record() v1.RecordServiceClient {
	return v1.NewRecordServiceClient(c.conn)
}

// Token is the root accessor for domain record related functions
func (c dnsClient) Token() v1.TokenServiceClient {
	return v1.NewTokenServiceClient(c.conn)
}

func (c Credentials) getTransportCredentials() (credentials.TransportCredentials, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system credentials: %w", err)
	}
	if c.CAFile == "" || c.Certfile == "" || c.Keyfile == "" || c.ServerName == "" {
		return nil, fmt.Errorf("all credentials properties must be configured")
	}
	ca, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("could not read ca certificate: %w", err)
	}

	ok := certPool.AppendCertsFromPEM(ca)
	if !ok {
		return nil, fmt.Errorf("failed to append ca certs: %s", c.CAFile)
	}

	clientCertificate, err := tls.LoadX509KeyPair(c.Certfile, c.Keyfile)
	if err != nil {
		return nil, fmt.Errorf("could not load client key pair: %w", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   c.ServerName,
		Certificates: []tls.Certificate{clientCertificate},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
	})
	return creds, nil
}

func (c ByteCredentials) getTransportCredentials() (credentials.TransportCredentials, error) {
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("failed to load system credentials: %w", err)
	}
	if string(c.CA) == "" || string(c.Cert) == "" || string(c.Key) == "" || c.ServerName == "" {
		return nil, fmt.Errorf("all credentials properties must be configured")
	}

	ok := certPool.AppendCertsFromPEM(c.CA)
	if !ok {
		return nil, fmt.Errorf("failed to append ca certs: %s", c.CA)
	}

	clientCertificate, err := tls.X509KeyPair(c.Cert, c.Key)
	if err != nil {
		return nil, fmt.Errorf("could not load client key pair: %w", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   c.ServerName,
		Certificates: []tls.Certificate{clientCertificate},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
	})
	return creds, nil
}
