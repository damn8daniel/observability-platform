.PHONY: build run test lint proto docker-build docker-up docker-down clean

BINARY_NAME=observability-platform
GO=go
DOCKER_COMPOSE=docker compose

## Build
build:
	$(GO) build -o bin/$(BINARY_NAME) ./cmd/server

run: build
	./bin/$(BINARY_NAME) -config config.yaml

## Testing
test:
	$(GO) test -v -race -count=1 ./...

test-coverage:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

## Linting
lint:
	golangci-lint run ./...

## Protobuf generation
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/observability.proto

## Docker
docker-build:
	docker build -t $(BINARY_NAME):latest .

docker-up:
	$(DOCKER_COMPOSE) up -d

docker-down:
	$(DOCKER_COMPOSE) down -v

docker-logs:
	$(DOCKER_COMPOSE) logs -f

## Dashboard
dashboard-install:
	cd web/dashboard && npm install

dashboard-dev:
	cd web/dashboard && npm start

dashboard-build:
	cd web/dashboard && npm run build

## Clean
clean:
	rm -rf bin/ coverage.out coverage.html
	$(DOCKER_COMPOSE) down -v

## Help
help:
	@echo "Available targets:"
	@echo "  build            - Build the Go binary"
	@echo "  run              - Build and run the server"
	@echo "  test             - Run tests with race detection"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  lint             - Run golangci-lint"
	@echo "  proto            - Regenerate protobuf code"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-up        - Start all services with Docker Compose"
	@echo "  docker-down      - Stop all services"
	@echo "  dashboard-dev    - Start React dashboard in dev mode"
	@echo "  dashboard-build  - Build React dashboard for production"
	@echo "  clean            - Remove build artifacts"
