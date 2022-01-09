package server

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/client"
	"github.com/majst01/metal-dns/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDomainCRUD(t *testing.T) {
	ctx := context.Background()
	pdns, err := test.StartPowerDNS()
	require.NoError(t, err)
	require.NotNil(t, pdns)

	addr, err := startGRPCServer(pdns)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// First create a connection which is only able to get a token
	clientConfig := client.DialConfig{
		Token:   "notokenforfirstreques",
		Address: &addr,
	}
	c, err := client.NewClient(ctx, clientConfig)
	require.NoError(t, err)
	require.NotNil(t, c)

	token, err := c.Token().Create(ctx,
		&v1.TokenCreateRequest{
			Issuer:  "Tester",
			Domains: []string{"example.com."},
			Permissions: []string{
				"/v1.TokenService/Create",
				"/v1.DomainService/Get",
				"/v1.DomainService/List",
				"/v1.DomainService/Create",
				"/v1.DomainService/Update",
				"/v1.DomainService/Delete",
				"/v1.RecordService/List",
				"/v1.RecordService/Create",
				"/v1.RecordService/Update",
				"/v1.RecordService/Delete",
			},
			Expires: durationpb.New(1 * time.Hour),
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Now create a new client connection with a token which can modify domains/records
	clientConfig = client.DialConfig{
		Token:   token.Token,
		Address: &addr,
	}
	c, err = client.NewClient(ctx, clientConfig)
	require.NoError(t, err)
	require.NotNil(t, c)

	ds, err := c.Domain().List(ctx, &v1.DomainsListRequest{Domains: []string{"example.com."}})
	require.NoError(t, err)
	require.Empty(t, ds.Domains)

	d1, err := c.Domain().Get(ctx, &v1.DomainGetRequest{Name: "example.com."})
	require.ErrorIs(t, err, status.Error(codes.NotFound, "Not Found"))
	require.Nil(t, d1)

	d1, err = c.Domain().Create(ctx, &v1.DomainCreateRequest{Name: "example.com.", Nameservers: []string{"ns1.example.com."}})
	require.NoError(t, err)
	require.NotNil(t, d1)

	require.Equal(t, "example.com.", d1.Domain.Name)
	require.NotNil(t, d1.Domain.Id)
	require.Equal(t, "example.com.", d1.Domain.Id)
	require.NotNil(t, d1.Domain.Url)

	d1, err = c.Domain().Delete(ctx, &v1.DomainDeleteRequest{Name: "example.com."})
	require.NoError(t, err)
	require.NotNil(t, d1)

	d1, err = c.Domain().Get(ctx, &v1.DomainGetRequest{Name: "example.com."})
	require.ErrorIs(t, err, status.Error(codes.NotFound, "Not Found"))
	require.Nil(t, d1)
}

func TestDomainService_List_DomainsFiltered(t *testing.T) {
	ctx := context.Background()
	pdns, err := test.StartPowerDNS()
	require.NoError(t, err)
	require.NotNil(t, pdns)

	addr, err := startGRPCServer(pdns)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// First create a connection which is only able to get a token
	clientConfig := client.DialConfig{
		Token:   "notokenforfirstrequest",
		Address: &addr,
	}
	c, err := client.NewClient(ctx, clientConfig)
	require.NoError(t, err)
	require.NotNil(t, c)

	token, err := c.Token().Create(ctx,
		&v1.TokenCreateRequest{
			Issuer:  "Tester",
			Domains: []string{"example.com.", "foo.bar."},
			Permissions: []string{
				"/v1.DomainService/List",
				"/v1.DomainService/Create",
			},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Now create a new client connection with a token which can modify domains/records
	clientConfig = client.DialConfig{
		Token:   token.Token,
		Address: &addr,
	}
	c, err = client.NewClient(ctx, clientConfig)
	require.NoError(t, err)
	require.NotNil(t, c)

	ds, err := c.Domain().List(ctx, &v1.DomainsListRequest{Domains: []string{"example.com."}})
	require.NoError(t, err)
	require.Empty(t, ds.Domains)

	d1, err := c.Domain().Create(ctx, &v1.DomainCreateRequest{Name: "example.com.", Nameservers: []string{"ns1.example.com."}})
	require.NoError(t, err)
	require.NotNil(t, d1)

	d2, err := c.Domain().Create(ctx, &v1.DomainCreateRequest{Name: "foo.bar.", Nameservers: []string{"ns1.foo.bar."}})
	require.NoError(t, err)
	require.NotNil(t, d2)

	// List only one domain
	ds, err = c.Domain().List(ctx, &v1.DomainsListRequest{Domains: []string{"example.com."}})
	require.NoError(t, err)
	require.NotEmpty(t, ds.Domains)
	require.Equal(t, []*v1.Domain{{Id: "example.com.", Name: "example.com.", Url: "/api/v1/servers/localhost/zones/example.com."}}, ds.Domains)

	// List wrong domain
	ds, err = c.Domain().List(ctx, &v1.DomainsListRequest{Domains: []string{"sample.com."}})
	require.ErrorIs(t, err, status.Error(codes.Unauthenticated, "domain:sample.com. is not allowed to list, only [example.com. foo.bar.]"))
	require.Nil(t, ds)

	// List without domain specified should returned allowed domains
	ds, err = c.Domain().List(ctx, &v1.DomainsListRequest{})
	require.NoError(t, err)
	require.NotEmpty(t, ds.Domains)
	require.Equal(t, []*v1.Domain{
		{Id: "example.com.", Name: "example.com.", Url: "/api/v1/servers/localhost/zones/example.com."},
		{Id: "foo.bar.", Name: "foo.bar.", Url: "/api/v1/servers/localhost/zones/foo.bar."},
	},
		ds.Domains)

}

func TestRecordCRUD(t *testing.T) {
	ctx := context.Background()
	pdns, err := test.StartPowerDNS()
	require.NoError(t, err)
	require.NotNil(t, pdns)

	addr, err := startGRPCServer(pdns)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// First create a connection which is only able to get a token
	clientConfig := client.DialConfig{
		Token:   "notokenforfirstrequest",
		Address: &addr,
	}
	c, err := client.NewClient(ctx, clientConfig)
	require.NoError(t, err)
	require.NotNil(t, c)

	token, err := c.Token().Create(ctx,
		&v1.TokenCreateRequest{
			Issuer:  "Tester",
			Domains: []string{"a.example.com."},
			Permissions: []string{
				"/v1.TokenService/Create",
				"/v1.DomainService/Get",
				"/v1.DomainService/List",
				"/v1.DomainService/Create",
				"/v1.DomainService/Update",
				"/v1.DomainService/Delete",
				"/v1.RecordService/List",
				"/v1.RecordService/Create",
				"/v1.RecordService/Update",
				"/v1.RecordService/Delete",
			},
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Now create a new client connection with a token which can modify domains/records
	clientConfig = client.DialConfig{
		Token:   token.Token,
		Address: &addr,
	}
	c, err = client.NewClient(ctx, clientConfig)
	require.NoError(t, err)
	require.NotNil(t, c)

	d1, err := c.Domain().Create(ctx, &v1.DomainCreateRequest{Name: "a.example.com.", Nameservers: []string{"ns1.example.com."}})
	require.NoError(t, err)
	require.NotNil(t, d1)

	r1, err := c.Record().Create(ctx, &v1.RecordCreateRequest{Type: v1.RecordType_A, Name: "www.a.example.com.", Data: "1.2.3.4", Ttl: uint32(600)})
	require.NoError(t, err)
	require.NotNil(t, r1)
	require.Equal(t, "www.a.example.com.", r1.Record.Name)
	require.Equal(t, "1.2.3.4", r1.Record.Data)

	addrs, err := pdns.Resolver.LookupHost(ctx, "www.a.example.com")
	require.NoError(t, err)
	require.NotNil(t, addrs)
	require.Contains(t, addrs, "1.2.3.4")

	r1, err = c.Record().Update(ctx, &v1.RecordUpdateRequest{Type: v1.RecordType_A, Name: "www.a.example.com.", Data: "2.3.4.5"})
	require.NoError(t, err)
	require.NotNil(t, r1)

	addrs, err = pdns.Resolver.LookupHost(ctx, "www.a.example.com")
	require.NoError(t, err)
	require.NotNil(t, addrs)
	require.Contains(t, addrs, "2.3.4.5")

	d1, err = c.Domain().Delete(ctx, &v1.DomainDeleteRequest{Name: "a.example.com."})
	require.NoError(t, err)
	require.NotNil(t, d1)

	d1, err = c.Domain().Get(ctx, &v1.DomainGetRequest{Name: "a.example.com."})
	require.ErrorIs(t, err, status.Error(codes.NotFound, "Not Found"))
	require.Nil(t, d1)
}

// Helper

func startGRPCServer(pdns *test.Pdns) (string, error) {
	log, _ := zap.NewProduction()

	config := DialConfig{
		Host:   "localhost",
		Port:   0,
		Secret: "secret",

		CA:      "../../certs/ca.pem",
		Cert:    "../../certs/server.pem",
		CertKey: "../../certs/server-key.pem",

		PdnsApiUrl:      pdns.BaseURL,
		PdnsApiPassword: pdns.APIKey,
		PdnsApiVHost:    pdns.VHost,
	}

	s, err := NewServer(log, config)
	if err != nil {
		return "", err
	}
	addr := s.listener.Addr().String()

	go func() {
		if err := s.Serve(); err != nil {
			panic(err)
		}
	}()

	if !isOpen(addr, 5*time.Second) {
		return "", fmt.Errorf("grpc server did not start within 5sec")
	}
	return addr, nil
}

func isOpen(addr string, timeout time.Duration) bool {
	var (
		conn net.Conn
		err  error
	)
	waited := 0 * time.Second
	for {
		conn, err = net.DialTimeout("tcp", addr, timeout)
		if err == nil {
			fmt.Printf("Error:%v", err)
			break
		}
		time.Sleep(1 * time.Second)
		waited = waited + 1*time.Second
		if waited > timeout {
			return false
		}
	}
	if conn != nil {
		conn.Close()
	}
	return true
}
