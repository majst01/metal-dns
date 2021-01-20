package main

import (
	"context"
	"os"
	"time"

	"github.com/majst01/metal-dns/pkg/auth"

	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/client"
	"go.uber.org/zap"
)

const grpcRequestTimeout = 5 * time.Second

func main() {

	logger, _ := zap.NewProduction()
	logger.Info("Starting Client")

	hmacKey := os.Getenv("HMAC_KEY")
	if hmacKey == "" {
		hmacKey = auth.HmacDefaultKey
	}

	c, err := client.NewClient(context.TODO(), "localhost", 50051, "certs/client.pem", "certs/client-key.pem", "certs/ca.pem", hmacKey, logger)
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
	log.Sugar().Infow("list records", "records", rs.Records)
	// create
	dcr := &v1.DomainCreateRequest{
		Name:        "a.example.com",
		Nameservers: []string{"localhost."},
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
