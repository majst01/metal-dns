FROM golang:1.19-alpine as builder

RUN apk add \
    gcc \
    git \
    libc-dev \
    make

WORKDIR /work
COPY . .
RUN make server

FROM alpine:3.17
COPY --from=builder /work/bin/server /
ENTRYPOINT [ "/server" ]