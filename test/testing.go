package test

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	pdnsOnce      sync.Once
	pdnsContainer testcontainers.Container
	pdnsApiKey    = "apipw"
)

type Pdns struct {
	BaseURL  string
	VHost    string
	APIKey   string
	Resolver *net.Resolver
}

func StartPowerDNS() (*Pdns, error) {
	ctx := context.Background()
	pdnsOnce.Do(func() {

		// --api=yes \
		// --api-key=apipw \

		var err error
		req := testcontainers.ContainerRequest{
			Image:        "powerdns/pdns-auth-46",
			ExposedPorts: []string{"80/tcp", "53/tcp"},
			WaitingFor: wait.ForAll(
				// wait.ForListeningPort("80/tcp"),
				wait.ForLog("[webserver] Listening for HTTP requests on 0.0.0.0:80"),
			),
			Cmd: []string{
				"--api=yes",
				"--api-key=" + pdnsApiKey,
				"--webserver=yes",
				"--webserver-address=0.0.0.0",
				"--webserver-port=80",
				"--webserver-allow-from=0.0.0.0/0",
				"--disable-syslog=yes",
				"--loglevel=9",
				"--log-dns-queries=yes",
				"--log-dns-details=yes",
				"--query-logging=yes",
			},
		}
		pdnsContainer, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: req,
			Started:          true,
		})
		if err != nil {
			panic(err.Error())
		}
	})
	ip, err := pdnsContainer.Host(ctx)
	if err != nil {
		return nil, err
	}
	apiport, err := pdnsContainer.MappedPort(ctx, "80")
	if err != nil {
		return nil, err
	}
	dnsport, err := pdnsContainer.MappedPort(ctx, "53")
	if err != nil {
		return nil, err
	}
	addr := fmt.Sprintf("http://%s:%d", ip, apiport.Int())

	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(1000),
			}
			return d.DialContext(ctx, "tcp", fmt.Sprintf("%s:%d", ip, dnsport.Int()))
		},
	}
	return &Pdns{
		BaseURL:  addr,
		VHost:    "localhost",
		APIKey:   pdnsApiKey,
		Resolver: r,
	}, err
}
