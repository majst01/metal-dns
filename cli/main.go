package main

import (
	"context"
	"os"
	"time"

	"github.com/bufbuild/connect-go"
	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/client"
	"go.uber.org/zap"
)

const grpcRequestTimeout = 5 * time.Second

func main() {

	logger, _ := zap.NewProduction()
	logger.Info("Starting Client")

	token := os.Getenv("JWT_TOKEN")
	if token == "" {
		// nolint:gosec
		token = "unknowntoken"
	}

	c := client.New(context.TODO(), client.DialConfig{
		Token:   token,
		BaseURL: "http://localhost:8080",
	})
	run(c, logger)

	logger.Info("Success")
}

func run(c client.Client, log *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), grpcRequestTimeout)
	defer cancel()

	token, err := c.Token().Create(ctx, connect.NewRequest(&v1.TokenServiceCreateRequest{
		Issuer: "John Doe",
		Domains: []string{
			"example.com.",
			"a.example.com.",
		},
		Permissions: []string{
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
	}))
	if err != nil {
		log.Error("could not create token", zap.Error(err))
	}
	log.Sugar().Infow("create token", "token", token)

	c = client.New(context.TODO(), client.DialConfig{
		Token:   token.Msg.Token,
		BaseURL: "http://localhost:8080",
	})

	ds, err := c.Domain().List(ctx, connect.NewRequest(&v1.DomainServiceListRequest{Domains: []string{"example.com."}}))
	if err != nil {
		log.Fatal("could not list domain", zap.Error(err))
	}
	log.Sugar().Infow("list domains", "domains", ds.Msg.Domains)

	rs, err := c.Record().List(ctx, connect.NewRequest(&v1.RecordServiceListRequest{Domain: "example.com."}))
	if err != nil {
		log.Fatal("could not list records", zap.Error(err))
	}

	log.Sugar().Infow("list records", "records", rs.Msg.Records)
	// create
	dcr := &v1.DomainServiceCreateRequest{
		Name:        "a.example.com.",
		Nameservers: []string{"ns1.example.com."},
	}
	d, err := c.Domain().Create(ctx, connect.NewRequest(dcr))
	if err != nil {
		log.Error("could not create domain", zap.Error(err))
	} else {
		log.Sugar().Infow("created domain", "domain", d.Msg.Domain)
	}

	// create record
	rcr := &v1.RecordServiceCreateRequest{
		Name: "www.a.example.com.",
		Type: v1.RecordType_A,
		Ttl:  3600,
		Data: "1.2.3.4",
	}
	r, err := c.Record().Create(ctx, connect.NewRequest(rcr))
	if err != nil {
		log.Error("could not create record", zap.Error(err))
	} else {
		log.Sugar().Infow("created record", "record", r.Msg.Record)
	}

	rlr := &v1.RecordServiceListRequest{Domain: "example.com.", Type: v1.RecordType_A}
	records, err := c.Record().List(ctx, connect.NewRequest(rlr))
	if err != nil {
		log.Error("could not list records", zap.Error(err))
	} else {
		log.Sugar().Infow("list records", "records", records.Msg.Records)
	}
}
