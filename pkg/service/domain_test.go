package service

import (
	"context"
	"testing"
	"time"

	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/test"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
)

func TestDomainListCreate(t *testing.T) {
	ctx := context.Background()
	pdns, err := test.StartPowerDNS()
	require.NoError(t, err)
	require.NotNil(t, pdns)

	log, _ := zap.NewProduction()

	ds := NewDomainService(log, pdns.BaseURL, pdns.VHost, pdns.APIKey, nil)
	require.NotNil(t, ds)

	rs := NewRecordService(log, pdns.BaseURL, pdns.VHost, pdns.APIKey, nil)
	require.NotNil(t, ds)

	token, err := newJWTToken("test", "Tester", []string{"example.com"}, nil, time.Hour, "secret")
	require.NoError(t, err)
	require.NotNil(t, token)

	ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{"authorization": "Bearer " + token}))

	domains, err := ds.List(ctx, &v1.DomainsListRequest{})
	require.NoError(t, err)
	require.NotNil(t, domains)

	z1, err := ds.Create(ctx, &v1.DomainCreateRequest{Name: "example.com.", Nameservers: []string{"ns1.example.com."}})
	require.NoError(t, err)
	require.NotNil(t, z1)
	require.Equal(t, "example.com.", z1.Domain.Name)
	require.NotNil(t, z1.Domain.Id)
	require.Equal(t, "example.com.", z1.Domain.Id)
	require.NotNil(t, z1.Domain.Url)

	ns, err := pdns.Resolver.LookupNS(ctx, "example.com.")
	require.NoError(t, err)
	require.NotNil(t, ns)
	require.GreaterOrEqual(t, 1, len(ns))
	require.Equal(t, "ns1.example.com.", ns[0].Host)

	r1, err := rs.Create(ctx, &v1.RecordCreateRequest{Type: v1.RecordType_A, Name: "www.example.com.", Data: "1.2.3.4", Ttl: uint32(600)})
	require.NoError(t, err)
	require.NotNil(t, r1)
	require.Equal(t, "www.example.com.", r1.Record.Name)
	require.Equal(t, "1.2.3.4", r1.Record.Data)

	rr, err := rs.List(ctx, &v1.RecordsListRequest{Domain: "example.com.", Type: v1.RecordType_A})
	require.NoError(t, err)
	require.NotNil(t, rr)
	require.Len(t, rr.Records, 1)
	require.Equal(t, "www.example.com.", rr.Records[0].Name)
	require.Equal(t, "1.2.3.4", rr.Records[0].Data)

	r1, err = rs.Update(ctx, &v1.RecordUpdateRequest{Type: v1.RecordType_A, Name: "www.example.com.", Data: "2.3.4.5", Ttl: uint32(300)})
	require.NoError(t, err)
	require.NotNil(t, r1)
	require.Equal(t, "www.example.com.", r1.Record.Name)
	require.Equal(t, "2.3.4.5", r1.Record.Data)
	require.Equal(t, uint32(300), r1.Record.Ttl)

	_, err = rs.Delete(ctx, &v1.RecordDeleteRequest{Type: v1.RecordType_A, Name: "www.example.com."})
	require.NoError(t, err)
	rr, err = rs.List(ctx, &v1.RecordsListRequest{Domain: "example.com.", Type: v1.RecordType_A})
	require.NoError(t, err)
	require.NotNil(t, rr)
	require.Len(t, rr.Records, 0)

	resp, err := ds.Delete(ctx, &v1.DomainDeleteRequest{Name: "example.com."})

	require.NoError(t, err)
	require.NotNil(t, resp)
}
