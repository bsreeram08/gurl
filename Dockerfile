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
    -ldflags="-s -w -X main.version=${VERSION} -X github.com/sreeram/gurl/internal/cli/commands.CurrentVersion=${VERSION}" \
    -o /out/gurl ./cmd/gurl

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S gurl \
    && adduser -S gurl -G gurl

USER gurl
WORKDIR /work

COPY --from=build /out/gurl /usr/local/bin/gurl

ENTRYPOINT ["gurl"]
CMD ["--help"]
