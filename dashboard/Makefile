.PHONY: help build release run dev clean test fmt vet lint tidy docker-build docker-run docker-stop docker-logs docker-shell

APP_NAME ?= dashboard
PKG ?= .
BIN_DIR ?= bin
DIST_DIR ?= dist
BINARY ?= $(BIN_DIR)/$(APP_NAME)
RELEASE_BINARY ?= $(DIST_DIR)/$(APP_NAME)-linux-amd64

GO ?= go
GOFLAGS ?=
LDFLAGS ?= -s -w

IMAGE_NAME ?= dashboard
IMAGE_TAG ?= latest
IMAGE ?= $(IMAGE_NAME):$(IMAGE_TAG)
DOCKERFILE ?= Dockerfile
CONTAINER_NAME ?= dashboard

SERVER_HOST ?= 0.0.0.0
PAAS_PORT ?= 3000
PAAS_ADMIN_USER ?= admin
PAAS_ADMIN_PASS ?= admin@123
DASHBOARD_AUTH_DISABLED ?= false
STACKS_DIR ?= /opt/stacks
BOLT_DB_FILE ?= $(STACKS_DIR)/.paas.db
DASHBOARD_DATA_FILE ?= data/dashboard.json

help:
	@echo "Available targets:"
	@echo "  build         - Build local development binary in ./$(BIN_DIR)"
	@echo "  release       - Build Linux amd64 release binary in ./$(DIST_DIR)"
	@echo "  run           - Build and run the app on the host"
	@echo "  dev           - Run with go run for development"
	@echo "  test          - Run the Go test suite"
	@echo "  fmt           - Format Go sources"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run fmt and vet"
	@echo "  tidy          - Tidy module files"
	@echo "  clean         - Remove build artifacts"
	@echo "  docker-build  - Build Docker image $(IMAGE)"
	@echo "  docker-run    - Run Docker container with stacks volume"
	@echo "  docker-stop   - Stop and remove Docker container"
	@echo "  docker-logs   - Tail Docker container logs"
	@echo "  docker-shell  - Open shell inside the running container"
	@echo ""
	@echo "Variables:"
	@echo "  IMAGE_NAME, IMAGE_TAG, CONTAINER_NAME"
	@echo "  PAAS_PORT, PAAS_ADMIN_USER, PAAS_ADMIN_PASS, DASHBOARD_AUTH_DISABLED"
	@echo "  STACKS_DIR, BOLT_DB_FILE, DASHBOARD_DATA_FILE"

$(BIN_DIR):
	mkdir -p $(BIN_DIR)

$(DIST_DIR):
	mkdir -p $(DIST_DIR)

build: $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -trimpath -ldflags="$(LDFLAGS)" -o $(BINARY) $(PKG)

release: $(DIST_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -trimpath -ldflags="$(LDFLAGS)" -o $(RELEASE_BINARY) $(PKG)

run: build
	SERVER_HOST=$(SERVER_HOST) \
	PAAS_PORT=$(PAAS_PORT) \
	PAAS_ADMIN_USER=$(PAAS_ADMIN_USER) \
	PAAS_ADMIN_PASS=$(PAAS_ADMIN_PASS) \
	DASHBOARD_AUTH_DISABLED=$(DASHBOARD_AUTH_DISABLED) \
	STACKS_DIR=$(STACKS_DIR) \
	BOLT_DB_FILE=$(BOLT_DB_FILE) \
	DASHBOARD_DATA_FILE=$(DASHBOARD_DATA_FILE) \
	$(BINARY)

dev:
	SERVER_HOST=$(SERVER_HOST) \
	PAAS_PORT=$(PAAS_PORT) \
	PAAS_ADMIN_USER=$(PAAS_ADMIN_USER) \
	PAAS_ADMIN_PASS=$(PAAS_ADMIN_PASS) \
	DASHBOARD_AUTH_DISABLED=$(DASHBOARD_AUTH_DISABLED) \
	STACKS_DIR=$(STACKS_DIR) \
	BOLT_DB_FILE=$(BOLT_DB_FILE) \
	DASHBOARD_DATA_FILE=$(DASHBOARD_DATA_FILE) \
	$(GO) run $(PKG)

test:
	$(GO) test ./...

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint: fmt vet

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR) $(DIST_DIR)
	$(GO) clean

docker-build:
	docker build -f $(DOCKERFILE) -t $(IMAGE) .

docker-run:
	mkdir -p $(STACKS_DIR)
	docker run -d \
		--name $(CONTAINER_NAME) \
		--restart unless-stopped \
		-p $(PAAS_PORT):3000 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(STACKS_DIR):/opt/stacks \
		-e SERVER_HOST=0.0.0.0 \
		-e PAAS_PORT=3000 \
		-e PAAS_ADMIN_USER=$(PAAS_ADMIN_USER) \
		-e PAAS_ADMIN_PASS=$(PAAS_ADMIN_PASS) \
		-e DASHBOARD_AUTH_DISABLED=$(DASHBOARD_AUTH_DISABLED) \
		-e STACKS_DIR=/opt/stacks \
		-e BOLT_DB_FILE=/opt/stacks/.paas.db \
		$(IMAGE)

docker-stop:
	-docker rm -f $(CONTAINER_NAME)

docker-logs:
	docker logs -f $(CONTAINER_NAME)

docker-shell:
	docker exec -it $(CONTAINER_NAME) /bin/bash
