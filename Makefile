SHELL := /bin/bash
APP_NAME := gpbackman
BRANCH_FULL := $(shell git rev-parse --abbrev-ref HEAD)
BRANCH := $(subst /,-,$(BRANCH_FULL))
GIT_REV := $(shell git describe --abbrev=7 --always)
ROOT_DIR := $(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
MUSL_CROSS := $(shell brew list| grep musl-cross)
UID := $(shell id -u)
GID := $(shell id -g)

.PHONY: test
test:
	@echo "Run tests for $(APP_NAME)"
	TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1  ./...

.PHONY: test-e2e
test-e2e:
	@echo "Run end-to-end tests for $(APP_NAME)"
	@make docker
	@make test-e2e_backup-info
	@make test-e2e_report-info
	@make test-e2e_backup-delete
	@make test-e2e_history-migrate

.PHONY: test-e2e_backup-info
test-e2e_backup-info:
	@echo "Run end-to-end tests for $(APP_NAME) for backup-info command"
	$(call down_docker_compose)
	$(call run_docker_compose,backup-info)
	$(call down_docker_compose)

.PHONY: test-e2e_backup-delete
test-e2e_backup-delete:
	@echo "Run end-to-end tests for $(APP_NAME) for backup-delete command"
	$(call down_docker_compose)
	$(call run_docker_compose,backup-delete)
	$(call down_docker_compose)

.PHONY: test-e2e_history-migrate
test-e2e_history-migrate:
	@echo "Run end-to-end tests for $(APP_NAME) for history-migrate command"
	$(call down_docker_compose)
	$(call run_docker_compose,history-migrate)
	$(call down_docker_compose)

.PHONY: test-e2e_report-info
test-e2e_report-info:
	@echo "Run end-to-end tests for $(APP_NAME) for report-info command"
	$(call down_docker_compose)
	$(call run_docker_compose,report-info)
	$(call down_docker_compose)

.PHONY: test-e2e-down
test-e2e-down:
	@echo "Stop old containers"
	$(call down_docker_compose)

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

.PHONY: docker
docker:
	@echo "Build $(APP_NAME) docker container"
	@echo "Version $(BRANCH)-$(GIT_REV)"
	DOCKER_BUILDKIT=1 docker build --pull -f Dockerfile --build-arg REPO_BUILD_TAG=$(BRANCH)-$(GIT_REV) -t $(APP_NAME) .

define e2e_command
	@echo "Run end-to-end tests for $(APP_NAME) for ${1} command"
	docker run --rm -v $(ROOT_DIR)/e2e_tests/:/home/gpbackman/e2e_tests --name="$(APP_NAME)" "$(APP_NAME)" /home/gpbackman/e2e_tests/run_e2e_${1}.sh
endef

define run_docker_compose
	GPBACKMAN_UID=$(UID) GPBACKMAN_GID=$(GID) docker-compose -f e2e_tests/docker-compose.yml run --rm --name ${1} ${1}
endef

define down_docker_compose
	GPBACKMAN_UID=$(UID) GPBACKMAN_GID=$(GID) docker-compose -f e2e_tests/docker-compose.yml down -v
endef