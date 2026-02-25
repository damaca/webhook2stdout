APP_NAME ?= webhook2stdout
BINARY ?= bin/$(APP_NAME)
GO ?= go
PORT ?= 8080
CONFIG ?= config.yaml

TAG ?= latest
GHCR_IMAGE ?= ghcr.io/$(shell echo "$${GITHUB_REPOSITORY:-local/$(APP_NAME)}" | tr '[:upper:]' '[:lower:]')
DOCKERHUB_IMAGE ?= $(APP_NAME)

.PHONY: help tidy fmt test build run clean docker-build docker-build-ghcr docker-build-dockerhub docker-push docker-push-ghcr docker-push-dockerhub docker-publish docker-run

help:
	@echo "Targets:"
	@echo "  make tidy               - Download/update Go modules"
	@echo "  make fmt                - Format Go code"
	@echo "  make test               - Run tests"
	@echo "  make build              - Build binary to $(BINARY)"
	@echo "  make run                - Run service locally"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make docker-build       - Build Docker image for both registries"
	@echo "  make docker-push        - Push Docker image to both registries"
	@echo "  make docker-publish     - Alias for docker-push"
	@echo "  make docker-run         - Run local Docker Hub-tagged image on port $(PORT)"

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

build:
	@mkdir -p $(dir $(BINARY))
	CGO_ENABLED=0 $(GO) build -o $(BINARY) .

run:
	$(GO) run . -config $(CONFIG)

clean:
	rm -rf bin

docker-build: docker-build-ghcr docker-build-dockerhub

docker-build-ghcr:
	docker build -t $(GHCR_IMAGE):$(TAG) .

docker-build-dockerhub:
	docker build -t $(DOCKERHUB_IMAGE):$(TAG) .

docker-push: docker-push-ghcr docker-push-dockerhub

docker-push-ghcr:
	docker push $(GHCR_IMAGE):$(TAG)

docker-push-dockerhub:
	docker push $(DOCKERHUB_IMAGE):$(TAG)

docker-publish: docker-push

docker-run:
	docker run --rm -p $(PORT):8080 $(DOCKERHUB_IMAGE):$(TAG)
