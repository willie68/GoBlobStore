##### BUILDER #####

FROM golang:1.17.6-alpine3.15 as builder

## Task: Install build deps

# hadolint ignore=DL3018
RUN set -eux; \
    apk add --no-progress --quiet --no-cache --upgrade --virtual .build-deps \
        gcc \
        git \
        musl-dev

## Task: copy source files

COPY . /src
WORKDIR /src
## Task: fetch project deps

RUN go mod download
RUN go mod tidy

## Task: build project

ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="0"

RUN go build -ldflags="-s -w" -o go-blob-store cmd/service/main.go 

## Task: set permissions

RUN chmod 0755 /src/go-blob-store

## Task: runtime dependencies

# hadolint ignore=DL3018
RUN set -eux; \
    apk add --no-progress --quiet --no-cache --upgrade --virtual .run-deps \
        tzdata

# hadolint ignore=DL3018,SC2183,DL4006
RUN set -eu +x; \
    apk add --no-progress --quiet --no-cache --upgrade ncurses; \
    apk update --quiet; \
    printf '%30s\n' | tr ' ' -; \
    echo "RUNTIME DEPENDENCIES"; \
    PKGNAME=$(apk info --depends .run-deps \
        | sed '/^$/d;/depends/d' \
        | sort -u ); \
    printf '%s\n' "${PKGNAME}" \
        | while IFS= read -r pkg; do \
                apk info --quiet --description --no-network "${pkg}" \
                | sed -n '/description/p' \
                | sed -r "s/($(echo "${pkg}" | sed -r 's/\+/\\+/g'))-(.*)\s.*/\1=\2/"; \
                done \
        | tee -a /usr/share/rundeps; \
    printf '%30s\n' | tr ' ' - 


##### TARGET #####
FROM alpine:3.15

ARG RELEASE
ENV IMG_VERSION="${RELEASE}"

COPY --from=builder /src/go-blob-store /usr/local/bin/
COPY --from=builder /src/configs/service.yaml /data/config/
COPY --from=builder /usr/share/rundeps /usr/share/rundeps

RUN set -eux; \
    xargs -a /usr/share/rundeps apk add --no-progress --quiet --no-cache --upgrade --virtual .run-deps

ENTRYPOINT ["/usr/local/bin/go-blob-store"]
CMD ["--config","/data/config/service.yaml"]

EXPOSE 8080 8443

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
  CMD wget -q -T 5 --spider http://localhost:8080/livez

LABEL org.opencontainers.image.title="go-blob-store" \
      org.opencontainers.image.description="MCS Blob storage service" \
      org.opencontainers.image.version="${IMG_VERSION}" \
      org.opencontainers.image.source="https://github.com/willie68/GoBlobStore.git" \
      org.opencontainers.image.vendor="MCS (www.rcarduino.de)" \
      org.opencontainers.image.authors="info@wk-music.de" \
      maintainer="MCS" \
      NAME="go-blob-store"

