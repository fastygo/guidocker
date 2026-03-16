# syntax=docker/dockerfile:1.7
#
# Build context: repository root (one level above gui-docker/)
#   docker build -f gui-docker/Dockerfile -t gui-docker:latest .

ARG GO_VERSION=1.22.2

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

# Copy module files first for better layer caching
COPY localdeps              ./localdeps
COPY ui8kit/go.mod           ./ui8kit/go.mod
COPY gui-docker/go.mod       ./gui-docker/go.mod
COPY gui-docker/go.sum       ./gui-docker/go.sum
COPY gui-docker/localdeps    ./gui-docker/localdeps

# Copy source trees
COPY ui8kit  ./ui8kit
COPY gui-docker ./gui-docker

WORKDIR /src/gui-docker

RUN CGO_ENABLED=0 go test ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/gui-docker ./cmd

# ---------------------------------------------------------------------------
FROM ubuntu:24.04

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    gnupg \
    iproute2 \
    git \
    tzdata \
    netcat-openbsd && \
    rm -rf /var/lib/apt/lists/*

RUN install -m 0755 -d /etc/apt/keyrings && \
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg | \
      gpg --dearmor -o /etc/apt/keyrings/docker.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
      https://download.docker.com/linux/ubuntu noble stable" \
      > /etc/apt/sources.list.d/docker.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends docker-ce-cli docker-compose-plugin && \
    rm -rf /var/lib/apt/lists/*

RUN if getent group docker >/dev/null 2>&1; then \
        groupmod -g 999 docker; \
    else \
        groupadd -g 999 docker; \
    fi && \
    useradd --system --create-home --home-dir /home/paas \
      --shell /usr/sbin/nologin --groups docker paas && \
    mkdir -p /opt/stacks /app/data && \
    chown -R paas:paas /opt/stacks /app /home/paas

WORKDIR /app

COPY --from=builder /out/gui-docker          /usr/local/bin/gui-docker
COPY gui-docker/scripts/docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
COPY gui-docker/data                         ./data
COPY --from=builder /src/gui-docker/static   ./static

ENV SERVER_HOST=0.0.0.0 \
    PAAS_PORT=3000 \
    PAAS_ADMIN_USER=admin \
    PAAS_ADMIN_PASS=admin@123 \
    STACKS_DIR=/opt/stacks \
    BOLT_DB_FILE=/opt/stacks/.paas.db \
    DASHBOARD_DATA_FILE=/app/data/dashboard.json

VOLUME ["/opt/stacks"]

EXPOSE 3000

HEALTHCHECK --interval=15s --timeout=5s --start-period=5s --retries=3 \
    CMD curl -sf http://localhost:3000/ || exit 1

RUN chmod +x /usr/local/bin/docker-entrypoint.sh

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
