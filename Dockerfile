# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.3-alpine AS build

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

RUN apk add --no-cache ca-certificates git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build \
    -ldflags="-s -w -X main.version=${VERSION} -X github.com/bsreeram08/valetkey/internal/cli/commands.CurrentVersion=${VERSION}" \
    -o /out/valetkey ./cmd/valetkey

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S valetkey \
    && adduser -S valetkey -G valetkey

COPY --from=build /out/valetkey /usr/local/bin/valetkey
# The pre-rename command name keeps working inside the image.
RUN ln -s valetkey /usr/local/bin/gurl

USER valetkey
WORKDIR /work

ENTRYPOINT ["valetkey"]
CMD ["--help"]
