.ONESHELL:
CGO_ENABLED := $(or ${CGO_ENABLED},0)
GO := go
GO111MODULE := on

SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},devel)

all: test

.PHONY: test
test:
	CGO_ENABLED=1 $(GO) test ./... -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out

.PHONY: protoc
protoc:
	docker run --rm --user $$(id -u):$$(id -g) -v ${PWD}:/work metalstack/builder protoc --proto_path=api --go_out=plugins=grpc:api api/v1/*.proto
	docker run --rm --user $$(id -u):$$(id -g) -v ${PWD}:/work metalstack/builder protoc --proto_path=api --go_out=plugins=grpc:api api/grpc/health/v1/*.proto

.PHONY: server
server:
	go build -tags netgo -ldflags "-X 'github.com/metal-stack/v.Version=$(VERSION)' \
								   -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
								   -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
								   -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'" \
						 -o bin/server main.go
	strip bin/server

.PHONY: client
client:
	go build -tags netgo -o bin/client cli/main.go
	strip bin/client



.PHONY: pdns-up
pdns-up: pdns-rm
	docker run -d --name powerdns -it --rm -p 8081:8081 -p 5533:53 powerdns/pdns-auth-44 --loglevel=5 --webserver=yes --webserver-address=0.0.0.0 --webserver-allow-from=0.0.0.0/0 --webserver-loglevel=detailed --api=yes --api-key=apipw
	docker exec -it powerdns pdnsutil create-zone example.com
	docker exec -it powerdns pdnsutil create-zone customera.example.com
	docker exec -it powerdns pdnsutil create-zone customerb.example.com
	docker exec -it powerdns pdnsutil add-record example.com www. A 1.2.3.4
	docker exec -it powerdns pdnsutil list-zone example.com

.PHONY: pdns-rm
pdns-rm:
	docker rm -f powerdns || true

.PHONY: certs
certs:
	cd certs && cfssl gencert -initca ca-csr.json | cfssljson -bare ca -
	cd certs && cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile client-server server.json | cfssljson -bare server -
	cd certs && cfssl gencert -ca=ca.pem -ca-key=ca-key.pem -config=ca-config.json -profile client client.json | cfssljson -bare client -
