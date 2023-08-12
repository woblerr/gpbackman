SHELL := /bin/bash
APP_NAME := gpbackman
BRANCH_FULL := $(shell git rev-parse --abbrev-ref HEAD)
BRANCH := $(subst /,-,$(BRANCH_FULL))
GIT_REV := $(shell git describe --abbrev=7 --always)
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
MUSL_CROSS := $(shell brew list| grep musl-cross)

.PHONY: test
test:
	@echo "Run tests for $(APP_NAME)"
	TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1  ./...

.PHONY: build
build:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags "-X main.version=$(BRANCH)-$(GIT_REV)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: build-on-darwin
build-on-darwin:
	@echo "Build $(APP_NAME)"
	@make test
	@if [ -z "$(MUSL_CROSS)" ]; then echo "musl-cross is not installed"; exit 1; fi;
	CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags "-linkmode external -extldflags -static -X main.version=$(BRANCH)-$(GIT_REV)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: build-darwin
build-darwin:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -mod=vendor -trimpath -ldflags "-X main.version=$(BRANCH)-$(GIT_REV)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: dist
dist:
	- @mkdir -p dist
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 -f Dockerfile.artifacts --progress=plain -t "$(APP_NAME)_dist" .
	- @docker rm -f "$(APP_NAME)_dist" 2>/dev/null || exit 0
	docker run -d --name="$(APP_NAME)_dist" "$(APP_NAME)_dist"
	docker cp "$(APP_NAME)_dist":/artifacts dist/
	docker rm -f "$(APP_NAME)_dist"
