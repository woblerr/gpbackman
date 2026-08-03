SHELL := /bin/bash
APP_NAME := gpbackman
BRANCH_FULL = $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null)
BRANCH = $(subst /,-,$(BRANCH_FULL))
GIT_REV = $(shell git describe --abbrev=7 --always 2>/dev/null)
VERSION ?= $(if $(GIT_REV),$(BRANCH)-$(GIT_REV),unknown)
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
MUSL_CROSS := $(shell brew list| grep musl-cross)
UID := $(shell id -u)
GID := $(shell id -g)
GPDB_CONTAINER_NAME := greenplum
GPDB_USER := gpadmin
E2E_DEFAULT_COMPOSE_FILE := e2e_tests/docker-compose.yml
E2E_STANDBY_COMPOSE_FILE := e2e_tests/docker-compose.standby.yml
E2E_STANDBY_COMMANDS := backup-delete backup-clean history-clean history-sync
# List of all e2e test commands
E2E_COMMANDS := backup-info report-info backup-delete backup-clean history-clean history-migrate history-sync

.PHONY: test
test:
	@echo "Run tests for $(APP_NAME)"
	TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1  ./...

.PHONY: lint
lint:
	@echo "Run linters for $(APP_NAME)"
	golangci-lint run ./...

.PHONY: fmt
fmt:
	@echo "Format Go files for $(APP_NAME)"
	go fmt ./...

define define_e2e_test
.PHONY: test-e2e_$(1)
test-e2e_$(1):
	@echo "Run end-to-end tests for $(APP_NAME) for $(1) command"
	$$(call down_docker_compose_all)
	$$(call run_docker_compose,$(1))
	$$(call run_e2e_tests,$(1))
	$$(call down_docker_compose_all)
endef

# Generate e2e test targets for all commands
$(foreach cmd,$(E2E_COMMANDS),$(eval $(call define_e2e_test,$(cmd))))

.PHONY: test-e2e
test-e2e:
	@for cmd in $(E2E_COMMANDS); do \
		echo "Running : $$cmd"; \
		$(MAKE) test-e2e_$$cmd || { echo "$$cmd failed."; exit 1; }; \
		echo "$$cmd passed"; \
	done

.PHONY: test-e2e-down
test-e2e-down:
	@echo "Stop old containers"
	$(call down_docker_compose_all)

.PHONY: build
build:
	@echo "Build $(APP_NAME)"
	CGO_ENABLED=1 go build -mod=vendor -trimpath -ldflags "-X main.version=$(VERSION)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: build-linux-on-darwin
build-linux-on-darwin:
	@echo "Build $(APP_NAME)"
	@make test
	@if [ -z "$(MUSL_CROSS)" ]; then echo "musl-cross is not installed"; exit 1; fi;
	CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags "-linkmode external -extldflags -static -X main.version=$(VERSION)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: build-darwin
build-darwin:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -mod=vendor -trimpath -ldflags "-X main.version=$(VERSION)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: build-linux
build-linux:
	@echo "Build $(APP_NAME)"
	@make test
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -mod=vendor -trimpath -ldflags "-X main.version=$(VERSION)" -o $(APP_NAME) $(APP_NAME).go

.PHONY: dist
dist:
	- @mkdir -p dist
	DOCKER_BUILDKIT=1 docker build -f Dockerfile.artifacts --progress=plain -t "$(APP_NAME)_dist" .
	- @docker rm -f "$(APP_NAME)_dist" 2>/dev/null || exit 0
	docker run -d --name="$(APP_NAME)_dist" "$(APP_NAME)_dist"
	docker cp "$(APP_NAME)_dist":/artifacts dist/
	docker rm -f "$(APP_NAME)_dist"

.PHONY: docker
docker:
	@echo "Build $(APP_NAME) docker container"
	@echo "Version $(VERSION)"
	DOCKER_BUILDKIT=1 docker build --pull -f Dockerfile --build-arg REPO_BUILD_TAG=$(VERSION) -t $(APP_NAME) .

.PHONY: docker-alpine
docker-alpine:
	@echo "Build $(APP_NAME) alpine docker container"
	@echo "Version $(VERSION)"
	DOCKER_BUILDKIT=1 docker build --pull -f Dockerfile.alpine --build-arg REPO_BUILD_TAG=$(VERSION) -t $(APP_NAME)-alpine .

define run_docker_compose
	@if [ -n "$(filter $(1),$(E2E_STANDBY_COMMANDS))" ]; then \
		chmod 600 e2e_tests/conf/ssh/id_rsa; \
		docker compose -f $(E2E_STANDBY_COMPOSE_FILE) up -d; \
	else \
		docker compose -f $(E2E_DEFAULT_COMPOSE_FILE) up -d; \
	fi
endef

define down_docker_compose_all
	docker compose -f $(E2E_STANDBY_COMPOSE_FILE) down -v --remove-orphans
	docker compose -f $(E2E_DEFAULT_COMPOSE_FILE) down -v --remove-orphans
endef

define run_e2e_tests
	docker exec "$(GPDB_CONTAINER_NAME)" su - ${GPDB_USER} -c "/home/$(GPDB_USER)/run_tests/run_test.sh $(1)"
endef
