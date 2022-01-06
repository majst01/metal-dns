# metal-dns

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
  "permissions:" [
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

`go get github.com/majst01/metal-dns`

```go
import (
  "context"
  "os"

  v1 "github.com/majst01/metal-dns/api/v1"
  "github.com/majst01/metal-dns/pkg/client"
)

func main() {
  c, err = client.NewClient(context.Background(), client.DialConfig{Token: os.Getenv("JWT_TOKEN")})
  if err != nil {
    panic(err)
  }

  dcr := &v1.DomainCreateRequest{
    Name:        "a.example.com.",
    Nameservers: []string{"ns1.example.com."},
  }
  d, err := c.Domain().Create(ctx, dcr)
  fmt.Println("Domain created:" + d)
}

```

## TODO

- implement health endpoint: https://stackoverflow.com/questions/59352845/how-to-implement-go-grpc-go-health-check