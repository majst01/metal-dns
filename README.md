# metal-dns

[![Actions](https://github.com/majst01/metal-dns/workflows/build/badge.svg)](https://github.com/majst01/metal-dns/actions)
[![GoDoc](https://pkg.go.dev/github.com/majst01/metal-dns?status.svg)](https://godoc.org/github.com/majst01/metal-dns)
[![Go Report Card](https://goreportcard.com/badge/github.com/majst01/metal-dns)](https://goreportcard.com/report/github.com/majst01/metal-dns)
[![codecov](https://codecov.io/gh/majst01/metal-dns/branch/master/graph/badge.svg)](https://codecov.io/gh/majst01/metal-dns)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/majst01/metal-dns/blob/master/LICENSE)

Acts as a authorization proxy in front of a powerdns resolver. Metal-DNS will restrict access to specific domains and subdomains.
Access to certain api actions can also be restricted.

A POC external-dns implementation is also available https://github.com/majst01/external-dns/tree/metal-dns-support .

Open Topics:

- Management of authorization tokens and who is able to modify certain domains.
- actually there is a Token create endpoint which can be used to create tokens with domains and permissions specified.

## Authorization

Standard JWT token authorization is implemented.

- get/list/create/update domains if not already present
- add/delete/update records

Example JWT Payload:

```json
{
  "sub": "1234567890",
  "name": "John Doe",
  "iat": 1516239022,
  "domains": [
     "a.example.com",
     "b.example.com"
  ],
  "permissions": [
    "/v1.DomainService/Get",
    "/v1.DomainService/List",
    "/v1.DomainService/Create",
    "/v1.DomainService/Update",
    "/v1.DomainService/Delete",
    "/v1.RecordService/Create",
    "/v1.RecordService/List",
    "/v1.RecordService/Update",
    "/v1.RecordService/Delete"
  ]
}
```

## Usage

### Server

1.) start Powerdns:

```bash
docker run -d --name powerdns --rm -p 8081:80 -p 5533:53 powerdns/pdns-auth-46 \
    --api=yes \
    --api-key=apipw \
    --webserver=yes \
    --webserver-address=0.0.0.0 \
    --webserver-port=80 \
    --webserver-allow-from=0.0.0.0/0 \
    --disable-syslog=yes \
    --loglevel=9 \
    --log-dns-queries=yes \
    --log-dns-details=yes \
    --query-logging=yes
```

2.) start metal-dns api server pointing to the powerdns api endpoint

```bash
make certs
docker run -d --name metal-dns --rm -p 50051:50051 -v $PWD/certs:/certs ghcr.io/majst01/metal-dns:main \
    --pdns-api-password=apipw \
    --pdns-api-url=http://localhost:8081 \
    --pdns-api-vhost=localhost \
    --secret=YOUR-JWT-TOKEN-SECRET
```

### Client

`go get github.com/majst01/metal-dns`

```go
import (
  "context"
  "os"

  v1 "github.com/majst01/metal-dns/api/v1"
  "github.com/majst01/metal-dns/pkg/client"
)

func main() {
  ctx := context.Background()
  dialConfig := client.DialConfig{
    Token: os.Getenv("JWT_TOKEN"),
  }
  c, err = client.NewClient(ctx, dialConfig)
  if err != nil {
    panic(err)
  }

  dcr := &v1.DomainCreateRequest{
    Name:        "a.example.com.",
    Nameservers: []string{"ns1.example.com."},
  }
  d, err := c.Domain().Create(ctx, dcr)
  if err != nil {
    panic(err)
  }
  fmt.Println("Domain created:" + d)

  rcr := &v1.RecordCreateRequest{
    Type: v1.RecordType_A,
    Name: "www.a.example.com.",
    Data: "1.2.3.4",
    Ttl: uint32(600),
  }

  r, err := c.Record().Create(ctx, rcr)
  if err != nil {
    panic(err)
  }
  fmt.Println("Record created:" + r)
}

```

## TODO

- implement health endpoint: https://stackoverflow.com/questions/59352845/how-to-implement-go-grpc-go-health-check