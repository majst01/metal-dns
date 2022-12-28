package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bufbuild/connect-go"
	grpchealth "github.com/bufbuild/connect-grpchealth-go"
	grpcreflect "github.com/bufbuild/connect-grpcreflect-go"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"github.com/majst01/metal-dns/api/v1/apiv1connect"
	"github.com/majst01/metal-dns/pkg/auth"
	"github.com/majst01/metal-dns/pkg/service"
	"github.com/metal-stack/v"
	"go.uber.org/zap"
)

type Server struct {
	log *zap.SugaredLogger
	c   DialConfig
}

type DialConfig struct {
	HttpServerEndpoint string
	Secret             string

	PdnsApiUrl      string
	PdnsApiPassword string
	PdnsApiVHost    string
}

func New(log *zap.SugaredLogger, config DialConfig) (*Server, error) {
	return &Server{
		log: log,
		c:   config,
	}, nil
}

func (s *Server) Serve() error {
	s.log.Infow("starting metal-dns", "version", v.V, "address", s.c.HttpServerEndpoint)

	authz, err := auth.NewOpaAuther(s.log, s.c.Secret)
	if err != nil {
		return fmt.Errorf("failed to create authorizer %w", err)
	}

	interceptors := connect.WithInterceptors(authz)

	domainService := service.NewDomainService(s.log, s.c.PdnsApiUrl, s.c.PdnsApiVHost, s.c.PdnsApiPassword, nil)
	recordService := service.NewRecordService(s.log, s.c.PdnsApiUrl, s.c.PdnsApiVHost, s.c.PdnsApiPassword, nil)
	tokenService := service.NewTokenService(s.log, s.c.Secret)

	mux := http.NewServeMux()

	// Register the services
	mux.Handle(apiv1connect.NewDomainServiceHandler(domainService, interceptors))
	mux.Handle(apiv1connect.NewRecordServiceHandler(recordService, interceptors))
	mux.Handle(apiv1connect.NewTokenServiceHandler(tokenService, interceptors))

	// Static HealthCheckers
	checker := grpchealth.NewStaticChecker(
		apiv1connect.DomainServiceName,
		apiv1connect.RecordServiceName,
		apiv1connect.TokenServiceName,
	)
	mux.Handle(grpchealth.NewHandler(checker))

	// enable remote service listing by enabling reflection
	reflector := grpcreflect.NewStaticReflector()
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	apiServer := &http.Server{
		Addr:              s.c.HttpServerEndpoint,
		Handler:           h2c.NewHandler(newCORS().Handler(mux), &http2.Server{}),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		MaxHeaderBytes:    8 * 1024, // 8KiB
	}
	s.log.Infof("serving http on %s", apiServer.Addr)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	go func() {
		if err := apiServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.log.Fatalf("HTTP listen and serve %v", err)
		}
	}()

	<-signals
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return apiServer.Shutdown(ctx)

}

func newCORS() *cors.Cors {
	// To let web developers play with the demo service from browsers, we need a
	// very permissive CORS setup.
	return cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowOriginFunc: func(origin string) bool {
			// Allow all origins, which effectively disables CORS.
			return true
		},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{
			// Content-Type is in the default safelist.
			"Accept",
			"Accept-Encoding",
			"Accept-Post",
			"Connect-Accept-Encoding",
			"Connect-Content-Encoding",
			"Connect-Protocol-Version",
			"Content-Encoding",
			"Grpc-Accept-Encoding",
			"Grpc-Encoding",
			"Grpc-Message",
			"Grpc-Status",
			"Grpc-Status-Details-Bin",
		},
		// Let browsers cache CORS information for longer, which reduces the number
		// of preflight requests. Any changes to ExposedHeaders won't take effect
		// until the cached data expires. FF caps this value at 24h, and modern
		// Chrome caps it at 2h.
		MaxAge: int(2 * time.Hour / time.Second),
	})
}
