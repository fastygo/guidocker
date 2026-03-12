# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.22.2

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

COPY go.mod ./
COPY localdeps ./localdeps
COPY . .

RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/dashboard .

FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

RUN useradd --system --create-home --home-dir /home/paas --shell /usr/sbin/nologin paas && \
    mkdir -p /opt/stacks /app/data && \
    chown -R paas:paas /opt/stacks /app /home/paas

WORKDIR /app

COPY --from=builder /out/dashboard /usr/local/bin/dashboard
COPY data ./data

ENV SERVER_HOST=0.0.0.0 \
    PAAS_PORT=3000 \
    PAAS_ADMIN_USER=admin \
    PAAS_ADMIN_PASS=admin@123 \
    STACKS_DIR=/opt/stacks \
    BOLT_DB_FILE=/opt/stacks/.paas.db \
    DASHBOARD_DATA_FILE=/app/data/dashboard.json

VOLUME ["/opt/stacks"]

EXPOSE 3000

USER paas

ENTRYPOINT ["/usr/local/bin/dashboard"]
