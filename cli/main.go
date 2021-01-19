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

	// create
	dcr := &v1.DomainCreateRequest{
		Name:      "test.example.com",
		Ipaddress: "127.0.0.1",
	}
	res, err := c.Domain().Create(ctx, dcr)
	if err != nil {
		log.Fatal("could not create prefix", zap.Error(err))
	}
	log.Sugar().Infow("created prefix", "prefix", res.Domain)
}
