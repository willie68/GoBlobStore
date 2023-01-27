##### BUILDER #####
FROM golang:1.18.3-alpine3.16 as builder

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
RUN go mod tidy -go=1.18

## Task: build project

ENV GOOS="linux"
ENV GOARCH="amd64"
ENV CGO_ENABLED="0"

##### TEST #####
FROM builder as test

# run unit tests with coverage
RUN chmod +x /src/build/unitentrytest.sh
RUN ls -l 
ENTRYPOINT [ "/src/build/unitentrytest.sh" ]
