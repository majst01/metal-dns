package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strings"

	"google.golang.org/grpc/reflection"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	apiv1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/service"

	"github.com/majst01/metal-dns/pkg/auth"
	"github.com/majst01/metal-dns/pkg/interceptors/grpc_internalerror"
	"github.com/metal-stack/v"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	moduleName = "metal-dns"
)

var (
	logger *zap.Logger
)

var rootCmd = &cobra.Command{
	Use:     moduleName,
	Short:   "an api manage dns for metal cloud components",
	Version: v.V.String(),
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("failed executing root command", zap.Error(err))
	}
}

func initConfig() {
	viper.SetEnvPrefix("DNS API")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().IntP("port", "", 50051, "the port to serve on")

	rootCmd.Flags().StringP("ca", "", "certs/ca.pem", "ca path")
	rootCmd.Flags().StringP("cert", "", "certs/server.pem", "server certificate path")
	rootCmd.Flags().StringP("certkey", "", "certs/server-key.pem", "server key path")

	// rootCmd.Flags().StringP("hmackey", "", auth.HmacDefaultKey, "preshared hmac key to authenticate.")

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		logger.Error("unable to construct root command", zap.Error(err))
	}
}

func run() {
	logger, _ = zap.NewProduction()
	defer func() {
		err := logger.Sync() // flushes buffer, if any
		if err != nil {
			fmt.Printf("unable to sync logger buffers:%v", err)
		}
	}()

	port := viper.GetInt("port")
	addr := fmt.Sprintf(":%d", port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		logger.Fatal("failed to listen", zap.Error(err))
	}

	logger.Info("starting metal-dns", zap.Stringer("version", v.V), zap.String("address", addr))

	caFile := viper.GetString("ca")
	// Get system certificate pool
	certPool, err := x509.SystemCertPool()
	if err != nil {
		logger.Fatal("could not read system certificate pool", zap.Error(err))
	}

	if caFile != "" {
		logger.Info("using ca", zap.String("ca", caFile))
		ca, err := os.ReadFile(caFile)
		if err != nil {
			logger.Fatal("could not read ca certificate", zap.Error(err))
		}
		// Append the certificates from the CA
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			logger.Fatal("failed to append ca certs", zap.Error(err))
		}
	}

	serverCert := viper.GetString("cert")
	serverKey := viper.GetString("certkey")
	cert, err := tls.LoadX509KeyPair(serverCert, serverKey)
	if err != nil {
		logger.Fatal("failed to load key pair", zap.Error(err))
	}

	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.NoClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	})

	authz, err := auth.NewOpaAuthorizer(auth.Logger(logger))
	if err != nil {
		logger.Fatal("failed to create authorizer", zap.Error(err))
	}

	opts := []grpc.ServerOption{
		// Enable TLS for all incoming connections.
		grpc.Creds(creds),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(logger),
			// grpc_auth.StreamServerInterceptor(auther.Auth),
			grpc_internalerror.StreamServerInterceptor(),
			grpc_recovery.StreamServerInterceptor(),
			authz.OpaStreamInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(logger),
			// grpc_auth.UnaryServerInterceptor(auther.Auth),
			grpc_internalerror.UnaryServerInterceptor(),
			grpc_recovery.UnaryServerInterceptor(),
			authz.OpaUnaryInterceptor,
		)),
	}

	// Set GRPC Interceptors
	// opts := []grpc.ServerOption{}
	// grpcServer := grpc.NewServer(opts...)
	grpcServer := grpc.NewServer(opts...)

	domainService := service.NewDomainService(logger, "http://localhost:8081", "localhost", "apipw", nil)
	recordService := service.NewRecordService(logger, "http://localhost:8081", "localhost", "apipw", nil)

	apiv1.RegisterDomainServiceServer(grpcServer, domainService)
	apiv1.RegisterRecordServiceServer(grpcServer, recordService)

	// After all your registrations, make sure all of the Prometheus metrics are initialized.
	grpc_prometheus.Register(grpcServer)
	// Register Prometheus metrics handler
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	go func() {
		logger.Info("starting metrics endpoint of :2112")
		err := http.ListenAndServe(":2112", metricsServer)
		if err != nil {
			logger.Error("failed to start metrics endpoint", zap.Error(err))
		}
		os.Exit(1)
	}()

	go func() {
		logger.Info("starting pprof endpoint of :2113")
		// inspect via
		// go tool pprof -http :8080 localhost:2113/debug/pprof/heap
		// go tool pprof -http :8080 localhost:2113/debug/pprof/goroutine
		err := http.ListenAndServe(":2113", nil)
		if err != nil {
			logger.Error("failed to start pprof endpoint", zap.Error(err))
		}
		os.Exit(1)
	}()

	reflection.Register(grpcServer)

	if err := grpcServer.Serve(lis); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}
}
