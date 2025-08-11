ARG REPO_BUILD_TAG="unknown"

FROM golang:1.24-bookworm AS builder
ARG REPO_BUILD_TAG
COPY . /build
WORKDIR /build
RUN apt-get update \
    && apt-get install -y --no-install-recommends build-essential \
    && CGO_ENABLED=1 go build \
        -mod=vendor -trimpath \
        -ldflags "-X main.version=${REPO_BUILD_TAG}" \
        -o gpbackman gpbackman.go \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

FROM ubuntu:24.04
ARG REPO_BUILD_TAG
ENV TZ="Etc/UTC" \
    GPBACKMAN_USER="gpbackman" \
    GPBACKMAN_GROUP="gpbackman" \
    GPBACKMAN_UID=1001 \
    GPBACKMAN_GID=1001
RUN apt-get update \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y \
        ca-certificates \
        gosu \
        tzdata \
    && groupadd --gid ${GPBACKMAN_GID} ${GPBACKMAN_GROUP} \
    && useradd --shell /bin/bash -d /home/${GPBACKMAN_USER} --uid ${GPBACKMAN_UID} --gid ${GPBACKMAN_GID} -m ${GPBACKMAN_USER} \
    && unlink /etc/localtime \
    && cp /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo "${TZ}" > /etc/timezone \
    && apt-get autoremove -y \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
COPY --chmod=755 docker_files/entrypoint.sh /entrypoint.sh
COPY --from=builder /build/gpbackman /usr/bin/gpbackman
LABEL \
    org.opencontainers.image.version="${REPO_BUILD_TAG}" \
    org.opencontainers.image.source="https://github.com/woblerr/gpbackman"
ENTRYPOINT ["/entrypoint.sh"]
CMD ["gpbackman", "-h"]
