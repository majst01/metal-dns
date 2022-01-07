package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"os"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"

	v1 "github.com/majst01/metal-dns/api/v1"
	"github.com/majst01/metal-dns/pkg/auth"
	"github.com/majst01/metal-dns/pkg/interceptors/grpc_internalerror"
	"github.com/majst01/metal-dns/pkg/service"
	"github.com/metal-stack/v"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	log      *zap.SugaredLogger
	listener net.Listener
	config   DialConfig
}

type DialConfig struct {
	Host   string
	Port   int
	Secret string

	CA      string
	Cert    string
	CertKey string

	PdnsApiUrl      string
	PdnsApiPassword string
	PdnsApiVHost    string
}

func NewServer(log *zap.Logger, config DialConfig) (*Server, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to listen %w", err)
	}
	return &Server{
		log:      log.Sugar(),
		listener: lis,
		config:   config,
	}, nil
}

func (s *Server) Serve() error {
	s.log.Infow("starting metal-dns", "version", v.V, "address", s.listener.Addr())

	caFile := s.config.CA
	// Get system certificate pool
	certPool, err := x509.SystemCertPool()
	if err != nil {
		return fmt.Errorf("could not read system certificate pool %w", err)
	}

	if caFile != "" {
		s.log.Infow("using ca", "ca", caFile)
		ca, err := os.ReadFile(caFile)
		if err != nil {
			return fmt.Errorf("could not read ca certificate %w", err)
		}
		// Append the certificates from the CA
		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			return fmt.Errorf("failed to append ca certs %w", err)
		}
	}

	cert, err := tls.LoadX509KeyPair(s.config.Cert, s.config.CertKey)
	if err != nil {
		return fmt.Errorf("failed to load key pair %w", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.NoClientCert,
		Certificates: []tls.Certificate{cert},
		ClientCAs:    certPool,
		MinVersion:   tls.VersionTLS12,
	})

	authz, err := auth.NewOpaAuther(s.log.Desugar(), s.config.Secret)
	if err != nil {
		return fmt.Errorf("failed to create authorizer %w", err)
	}

	opts := []grpc.ServerOption{
		// Enable TLS for all incoming connections.
		grpc.Creds(creds),
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_ctxtags.StreamServerInterceptor(),
			grpc_prometheus.StreamServerInterceptor,
			grpc_zap.StreamServerInterceptor(s.log.Desugar()),
			grpc_internalerror.StreamServerInterceptor(),
			grpc_recovery.StreamServerInterceptor(),
			authz.OpaStreamInterceptor,
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_prometheus.UnaryServerInterceptor,
			grpc_zap.UnaryServerInterceptor(s.log.Desugar()),
			grpc_internalerror.UnaryServerInterceptor(),
			grpc_recovery.UnaryServerInterceptor(),
			authz.OpaUnaryInterceptor,
		)),
	}

	// Set GRPC Interceptors
	// opts := []grpc.ServerOption{}
	// grpcServer := grpc.NewServer(opts...)
	grpcServer := grpc.NewServer(opts...)

	domainService := service.NewDomainService(s.log.Desugar(), s.config.PdnsApiUrl, s.config.PdnsApiVHost, s.config.PdnsApiPassword, nil)
	recordService := service.NewRecordService(s.log.Desugar(), s.config.PdnsApiUrl, s.config.PdnsApiVHost, s.config.PdnsApiPassword, nil)
	tokenService := service.NewTokenService(s.log.Desugar(), s.config.Secret)

	v1.RegisterDomainServiceServer(grpcServer, domainService)
	v1.RegisterRecordServiceServer(grpcServer, recordService)
	v1.RegisterTokenServiceServer(grpcServer, tokenService)

	// After all your registrations, make sure all of the Prometheus metrics are initialized.
	grpc_prometheus.Register(grpcServer)
	reflection.Register(grpcServer)

	// Register Health Service
	grpc_health_v1.RegisterHealthServer(grpcServer, domainService)

	return grpcServer.Serve(s.listener)
}

func (s *Server) StartMetricsAndPprof() {
	// Register Prometheus metrics handler
	metricsServer := http.NewServeMux()
	metricsServer.Handle("/metrics", promhttp.Handler())
	go func() {
		s.log.Info("starting metrics endpoint of :2112")
		err := http.ListenAndServe(":2112", metricsServer)
		if err != nil {
			s.log.Errorw("failed to start metrics endpoint", "error", err)
		}
		os.Exit(1)
	}()

	go func() {
		s.log.Info("starting pprof endpoint of :2113")
		// inspect via
		// go tool pprof -http :8080 localhost:2113/debug/pprof/heap
		// go tool pprof -http :8080 localhost:2113/debug/pprof/goroutine
		err := http.ListenAndServe(":2113", nil)
		if err != nil {
			s.log.Errorw("failed to start pprof endpoint", "error", err)
		}
		os.Exit(1)
	}()
}
