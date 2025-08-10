ARG REPO_BUILD_TAG="unknown"

FROM golang:1.24-alpine3.21 AS builder
ARG REPO_BUILD_TAG
COPY . /build
WORKDIR /build
RUN apk add --no-cache --update build-base \
    && CGO_ENABLED=1 go build \
        -mod=vendor -trimpath \
        -ldflags "-X main.version=${REPO_BUILD_TAG}" \
        -o gpbackman gpbackman.go

FROM alpine:3.21
ARG REPO_BUILD_TAG
ENV TZ="Etc/UTC" \
    GPBACKMAN_USER="gpbackman" \
    GPBACKMAN_UID=1001 \
    GPBACKMAN_GID=1001
RUN apk add --no-cache --update \
        ca-certificates \
        su-exec \
        tzdata \
    && adduser -s /bin/sh -h /home/$GPBACKMAN_USER -D -u $GPBACKMAN_UID $GPBACKMAN_USER \
    && cp /usr/share/zoneinfo/${TZ} /etc/localtime \
    && echo "${TZ}" > /etc/timezone \
    && rm -rf /var/cache/apk/*
COPY --chmod=755 docker_files/entrypoint.sh /entrypoint.sh
COPY --from=builder /build/gpbackman /usr/bin/gpbackman
LABEL \
    org.opencontainers.image.version="${REPO_BUILD_TAG}" \
    org.opencontainers.image.source="https://github.com/woblerr/gpbackman"
ENTRYPOINT ["/entrypoint.sh"]
CMD ["gpbackman", "-h"]
