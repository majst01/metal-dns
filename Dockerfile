FROM golang:1.20-alpine as builder

RUN apk add \
    gcc \
    git \
    libc-dev \
    make

WORKDIR /work
COPY . .
RUN make server

FROM alpine:3.18
COPY --from=builder /work/bin/server /
ENTRYPOINT [ "/server" ]