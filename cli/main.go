package main

import (
	"context"
	"os"
	"time"

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

	c, err := client.NewClient(context.TODO(), client.DialConfig{Token: token})
	if err != nil {
		logger.Fatal(err.Error())
	}
	defer func() {
		err = c.Close()
		if err != nil {
			logger.Fatal(err.Error())
		}
	}()
	run(c, logger)

	logger.Info("Success")
}

func run(c client.Client, log *zap.Logger) {
	ctx, cancel := context.WithTimeout(context.Background(), grpcRequestTimeout)
	defer cancel()

	token, err := c.Token().Create(ctx, &v1.TokenCreateRequest{
		Issuer:  "John Doe",
		Domains: []string{"example.com."},
		Permissions: []string{
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
	})
	if err != nil {
		log.Error("could not create token", zap.Error(err))
	}
	log.Sugar().Infow("create token", "token", token)

	c, err = client.NewClient(context.TODO(), client.DialConfig{Token: token.Token})
	if err != nil {
		log.Fatal(err.Error())
	}

	ds, err := c.Domain().List(ctx, &v1.DomainsListRequest{Domains: []string{"example.com."}})
	if err != nil {
		log.Fatal("could not list domain", zap.Error(err))
	}
	log.Sugar().Infow("list domains", "domains", ds.Domains)

	rs, err := c.Record().List(ctx, &v1.RecordsListRequest{Domain: "example.com."})
	if err != nil {
		log.Fatal("could not list records", zap.Error(err))
	}

	log.Sugar().Infow("list records", "records", rs.Records)
	// create
	dcr := &v1.DomainCreateRequest{
		Name:        "a.example.com.",
		Nameservers: []string{"ns1.example.com."},
	}
	d, err := c.Domain().Create(ctx, dcr)
	if err != nil {
		log.Error("could not create domain", zap.Error(err))
	} else {
		log.Sugar().Infow("created domain", "domain", d.Domain)
	}

	// create record
	rcr := &v1.RecordCreateRequest{
		Name: "www.a.example.com.",
		Type: v1.RecordType_A,
		Ttl:  3600,
		Data: "1.2.3.4",
	}
	r, err := c.Record().Create(ctx, rcr)
	if err != nil {
		log.Error("could not create record", zap.Error(err))
	} else {
		log.Sugar().Infow("created record", "record", r.Record)
	}

	rlr := &v1.RecordsListRequest{Domain: "example.com.", Type: v1.RecordType_A}
	records, err := c.Record().List(ctx, rlr)
	if err != nil {
		log.Error("could not list records", zap.Error(err))
	} else {
		log.Sugar().Infow("list records", "records", records.Records)
	}
}
