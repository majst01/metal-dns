# metal-dns

[![Actions](https://github.com/majst01/metal-dns/workflows/build/badge.svg)](https://github.com/majst01/metal-dns/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/majst01/metal-dns.svg)](https://pkg.go.dev/github.com/majst01/metal-dns)
[![Go Report Card](https://goreportcard.com/badge/github.com/majst01/metal-dns)](https://goreportcard.com/report/github.com/majst01/metal-dns)
[![codecov](https://codecov.io/gh/majst01/metal-dns/branch/master/graph/badge.svg)](https://codecov.io/gh/majst01/metal-dns)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/majst01/metal-dns/blob/master/LICENSE)

Acts as a authorization proxy in front of a powerdns resolver. Metal-DNS will restrict access to specific domains and subdomains.
Access to certain api actions can also be restricted.

A POC external-dns implementation is also available <https://github.com/majst01/external-dns/tree/metal-dns-support> .

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
docker run -d --rm \
  --name powerdns \
  -p 8081:80 \
  -p 5533:53 powerdns/pdns-auth-46 \
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
docker run -d --rm \
  --name metal-dns \
  -p 50051:50051 \
  -v $PWD/certs:/certs ghcr.io/majst01/metal-dns \
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
  addr := "localhost:50051"
  dialConfig := client.DialConfig{
    Address: &addr,
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

## Onboarding

In order to make this application useful, a self service onboarding and token generation must be implemented.

The proposal would be the following:

1.) The user will login with github authentication against a login endpoint ( [sample](https://github.com/dghubble/gologin/tree/master/examples/github) )

2.) This endpoint will then offer:

- a list of available domains or subdomains to acquire.
- acquire domain or subdomain
- transfer a already owned domain ( future feature )
- create API Token for all or only a subset of acquired domains, with permissions and expiration
- remove API Tokens
- release acquired domains

3.) The backend stores:

- a list of available domains/subdomains
- a mapping from username to acquired domains
- a mapping from username to created tokens

### Domains and Subdomains

To ease the process of register dns entries for specific services, metal-dns adds the ability to delegate subdomains to individual users.
This is actually not possible at prominent cloud providers, where a user must register a domain and can then act on the whole domain.

So for example we could offer all subdomains below `metal-dns.org`. A user can the acquire `<username>.metal-dns.org` and then add whatever host or subdomain he wants below that.
It should also be possible to acquire a random subdomain like `myshop.metal-dns.org`. Only the first subdomain below one of the available domains are able to be acquired.
