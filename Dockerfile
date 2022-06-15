FROM golang:1.18-alpine as builder

RUN apk add \
    gcc \
    libc-dev \
    make

WORKDIR /work
COPY . .
RUN make certs server

FROM alpine:3.16
COPY --from=builder /work/bin/server /
ENTRYPOINT [ "/server" ]