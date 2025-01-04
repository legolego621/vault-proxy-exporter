# build golang application
FROM golang:1.23.1 AS builder

WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -o vault-proxy-exporter /build/cmd/vault-proxy-exporter

# build docker image
FROM alpine:3.21.0

ARG NAME=vault-proxy-exporter
ARG VERSION=latest

RUN apk add --no-cache \
  curl \
  && rm -rf /var/cache/apk/*

RUN addgroup ${NAME} && adduser -S -G ${NAME} ${NAME}

ENV HOME=/home/${NAME}

USER ${NAME}

COPY --from=builder /build/${NAME} /bin/${NAME}

CMD ["/bin/vault-proxy-exporter"]
