package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bufbuild/connect-go"
	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/api/v1/apiv1connect"

	"github.com/majst01/metal-dns/pkg/auth"
	"github.com/majst01/metal-dns/pkg/client"
	"github.com/majst01/metal-dns/pkg/service"
	"github.com/majst01/metal-dns/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/types/known/durationpb"
)

func TestDomainCRUD(t *testing.T) {
	ctx := context.Background()
	pdns, err := test.StartPowerDNS()
	require.NoError(t, err)
	require.NotNil(t, pdns)

	addr, err := startGRPCServer(t, pdns)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// First create a connection which is only able to get a token
	clientConfig := client.DialConfig{
		Token:   "notokenforfirstrequest",
		BaseURL: addr,
	}
	c := client.New(ctx, clientConfig)
	require.NotNil(t, c)

	jwttoken, err := c.Token().Create(ctx,
		connect.NewRequest(&v1.TokenServiceCreateRequest{
			Issuer:  "Tester",
			Domains: []string{"example.com."},
			Permissions: []string{
				"/api.v1.TokenService/Create",
				"/api.v1.DomainService/Get",
				"/api.v1.DomainService/List",
				"/api.v1.DomainService/Create",
				"/api.v1.DomainService/Update",
				"/api.v1.DomainService/Delete",
				"/api.v1.RecordService/Get",
				"/api.v1.RecordService/List",
				"/api.v1.RecordService/Create",
				"/api.v1.RecordService/Update",
				"/api.v1.RecordService/Delete",
			},
			Expires: durationpb.New(1 * time.Hour),
		},
		))
	require.NoError(t, err)
	require.NotEmpty(t, jwttoken)

	// Now create a new client connection with a token which can modify domains/records
	clientConfig = client.DialConfig{
		Token:   jwttoken.Msg.Token,
		BaseURL: addr,
	}
	c = client.New(ctx, clientConfig)
	require.NotNil(t, c)

	ds, err := c.Domain().List(ctx, connect.NewRequest(&v1.DomainServiceListRequest{Domains: []string{"example.com."}}))
	require.NoError(t, err)
	require.Empty(t, ds.Msg.Domains)

	d1, err := c.Domain().Get(ctx, connect.NewRequest(&v1.DomainServiceGetRequest{Name: "example.com."}))
	require.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
	require.Nil(t, d1)

	d2, err := c.Domain().Create(ctx, connect.NewRequest(&v1.DomainServiceCreateRequest{Name: "example.com.", Nameservers: []string{"ns1.example.com."}}))
	require.NoError(t, err)
	require.NotNil(t, d2)

	require.Equal(t, "example.com.", d2.Msg.Domain.Name)
	require.NotNil(t, d2.Msg.Domain.Id)
	require.Equal(t, "example.com.", d2.Msg.Domain.Id)
	require.NotNil(t, d2.Msg.Domain.Url)

	d3, err := c.Domain().Delete(ctx, connect.NewRequest(&v1.DomainServiceDeleteRequest{Name: "example.com."}))
	require.NoError(t, err)
	require.NotNil(t, d3)

	d4, err := c.Domain().Get(ctx, connect.NewRequest(&v1.DomainServiceGetRequest{Name: "example.com."}))
	require.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
	require.Nil(t, d4)
}

func TestDomainService_List_DomainsFiltered(t *testing.T) {
	ctx := context.Background()
	pdns, err := test.StartPowerDNS()
	require.NoError(t, err)
	require.NotNil(t, pdns)

	addr, err := startGRPCServer(t, pdns)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// First create a connection which is only able to get a token
	clientConfig := client.DialConfig{
		Token:   "notokenforfirstrequest",
		BaseURL: addr,
	}
	c := client.New(ctx, clientConfig)
	require.NotNil(t, c)

	token, err := c.Token().Create(ctx,
		connect.NewRequest(&v1.TokenServiceCreateRequest{
			Issuer:  "Tester",
			Domains: []string{"example.com.", "foo.bar."},
			Permissions: []string{
				"/api.v1.DomainService/List",
				"/api.v1.DomainService/Create",
			},
		},
		))
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Now create a new client connection with a token which can modify domains/records
	clientConfig = client.DialConfig{
		Token:   token.Msg.Token,
		BaseURL: addr,
	}
	c = client.New(ctx, clientConfig)
	require.NotNil(t, c)

	ds, err := c.Domain().List(ctx, connect.NewRequest(&v1.DomainServiceListRequest{Domains: []string{"example.com."}}))
	require.NoError(t, err)
	require.Empty(t, ds.Msg.Domains)

	d1, err := c.Domain().Create(ctx, connect.NewRequest(&v1.DomainServiceCreateRequest{Name: "example.com.", Nameservers: []string{"ns1.example.com."}}))
	require.NoError(t, err)
	require.NotNil(t, d1)

	d2, err := c.Domain().Create(ctx, connect.NewRequest(&v1.DomainServiceCreateRequest{Name: "foo.bar.", Nameservers: []string{"ns1.foo.bar."}}))
	require.NoError(t, err)
	require.NotNil(t, d2)

	// List only one domain
	ds2, err := c.Domain().List(ctx, connect.NewRequest(&v1.DomainServiceListRequest{Domains: []string{"example.com."}}))
	require.NoError(t, err)
	require.NotEmpty(t, ds2.Msg.Domains)
	require.Equal(t, []*v1.Domain{{Id: "example.com.", Name: "example.com.", Url: "/api/v1/servers/localhost/zones/example.com."}}, ds2.Msg.Domains)

	// List wrong domain
	ds, err = c.Domain().List(ctx, connect.NewRequest(&v1.DomainServiceListRequest{Domains: []string{"sample.com."}}))
	require.Equal(t, connect.CodeOf(err), connect.CodeUnauthenticated)
	require.Equal(t, err.Error(), "unauthenticated: domain:sample.com. is not allowed to list, only [example.com. foo.bar.]")
	require.Nil(t, ds)

	// List without domain specified should returned allowed domains
	ds, err = c.Domain().List(ctx, connect.NewRequest(&v1.DomainServiceListRequest{}))
	require.NoError(t, err)
	require.NotEmpty(t, ds.Msg.Domains)
	require.Equal(t, []*v1.Domain{
		{Id: "example.com.", Name: "example.com.", Url: "/api/v1/servers/localhost/zones/example.com."},
		{Id: "foo.bar.", Name: "foo.bar.", Url: "/api/v1/servers/localhost/zones/foo.bar."},
	}, ds.Msg.Domains)

}

func TestRecordCRUD(t *testing.T) {
	ctx := context.Background()
	pdns, err := test.StartPowerDNS()
	require.NoError(t, err)
	require.NotNil(t, pdns)

	addr, err := startGRPCServer(t, pdns)
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	// First create a connection which is only able to get a token
	clientConfig := client.DialConfig{
		Token:   "notokenforfirstrequest",
		BaseURL: addr,
	}
	c := client.New(ctx, clientConfig)
	require.NotNil(t, c)

	token, err := c.Token().Create(ctx,
		connect.NewRequest(&v1.TokenServiceCreateRequest{
			Issuer:  "Tester",
			Domains: []string{"a.example.com."},
			Permissions: []string{
				"/api.v1.TokenService/Create",
				"/api.v1.DomainService/Get",
				"/api.v1.DomainService/List",
				"/api.v1.DomainService/Create",
				"/api.v1.DomainService/Update",
				"/api.v1.DomainService/Delete",
				"/api.v1.RecordService/List",
				"/api.v1.RecordService/Create",
				"/api.v1.RecordService/Update",
				"/api.v1.RecordService/Delete",
			},
		},
		))
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Now create a new client connection with a token which can modify domains/records
	clientConfig = client.DialConfig{
		Token:   token.Msg.Token,
		BaseURL: addr,
	}
	c = client.New(ctx, clientConfig)
	require.NotNil(t, c)

	d1, err := c.Domain().Create(ctx, connect.NewRequest(&v1.DomainServiceCreateRequest{Name: "a.example.com.", Nameservers: []string{"ns1.example.com."}}))
	require.NoError(t, err)
	require.NotNil(t, d1)

	r1, err := c.Record().Create(ctx, connect.NewRequest(&v1.RecordServiceCreateRequest{Type: v1.RecordType_A, Name: "www.a.example.com.", Data: "1.2.3.4", Ttl: uint32(600)}))
	require.NoError(t, err)
	require.NotNil(t, r1)
	require.Equal(t, "www.a.example.com.", r1.Msg.Record.Name)
	require.Equal(t, "1.2.3.4", r1.Msg.Record.Data)

	addrs, err := pdns.Resolver.LookupHost(ctx, "www.a.example.com")
	require.NoError(t, err)
	require.NotNil(t, addrs)
	require.Contains(t, addrs, "1.2.3.4")

	r2, err := c.Record().Update(ctx, connect.NewRequest(&v1.RecordServiceUpdateRequest{Type: v1.RecordType_A, Name: "www.a.example.com.", Data: "2.3.4.5"}))
	require.NoError(t, err)
	require.NotNil(t, r2)

	addrs, err = pdns.Resolver.LookupHost(ctx, "www.a.example.com")
	require.NoError(t, err)
	require.NotNil(t, addrs)
	require.Contains(t, addrs, "2.3.4.5")

	d2, err := c.Domain().Delete(ctx, connect.NewRequest(&v1.DomainServiceDeleteRequest{Name: "a.example.com."}))
	require.NoError(t, err)
	require.NotNil(t, d2)

	d3, err := c.Domain().Get(ctx, connect.NewRequest(&v1.DomainServiceGetRequest{Name: "a.example.com."}))
	require.Equal(t, connect.CodeOf(err), connect.CodeNotFound)
	require.Nil(t, d3)
}

// Helper

func startGRPCServer(t *testing.T, pdns *test.Pdns) (string, error) {
	log := zaptest.NewLogger(t).Sugar()

	config := DialConfig{
		PdnsApiUrl:      pdns.BaseURL,
		PdnsApiPassword: pdns.APIKey,
		PdnsApiVHost:    pdns.VHost,
	}

	mux := http.NewServeMux()

	authz, err := auth.NewOpaAuther(log, "secret")
	if err != nil {
		return "", fmt.Errorf("failed to create authorizer %w", err)
	}
	interceptors := connect.WithInterceptors(authz)

	domainService := service.NewDomainService(log, config.PdnsApiUrl, config.PdnsApiVHost, config.PdnsApiPassword, nil)
	recordService := service.NewRecordService(log, config.PdnsApiUrl, config.PdnsApiVHost, config.PdnsApiPassword, nil)
	tokenService := service.NewTokenService(log, "secret")

	mux.Handle(apiv1connect.NewDomainServiceHandler(domainService, interceptors))
	mux.Handle(apiv1connect.NewRecordServiceHandler(recordService, interceptors))
	mux.Handle(apiv1connect.NewTokenServiceHandler(tokenService, interceptors))

	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.Start()

	return server.URL, nil
}
