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
		token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJuYmYiOjE1MTYyMzkwMjIsImRvbWFpbnMiOlsiYS5leGFtcGxlLmNvbSIsImIuZXhhbXBsZS5jb20iXX0.jPEP4TKNpmAcDz_y6AK3wtDr6UOpE69dAylp_qwUNGU"
	}

	c, err := client.NewClient(context.TODO(), "localhost", 50051, "certs/client.pem", "certs/client-key.pem", "certs/ca.pem", token, logger)
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
	ds, err := c.Domain().List(ctx, &v1.DomainsListRequest{})
	if err != nil {
		log.Error("could not create domain", zap.Error(err))
	}
	log.Sugar().Infow("list domains", "domains", ds.Domains)

	rs, err := c.Record().List(ctx, &v1.RecordsListRequest{Domain: "example.com"})
	if err != nil {
		log.Error("could not list records", zap.Error(err))
	}

	if true {
		return
	}
	log.Sugar().Infow("list records", "records", rs.Records)
	// create
	dcr := &v1.DomainCreateRequest{
		Name:        "a.example.com",
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
		Name: "www.a.example.com",
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

	rlr := &v1.RecordsListRequest{Domain: "example.com", Type: v1.RecordType_A}
	records, err := c.Record().List(ctx, rlr)
	if err != nil {
		log.Error("could not list records", zap.Error(err))
	} else {
		log.Sugar().Infow("list records", "records", records.Records)
	}
}
