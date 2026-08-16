# syntax=docker/dockerfile:1
ARG GO_VERSION=1.26

# Build on the native platform and cross-compile, which is far cheaper than
# emulating the target platform under qemu.
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

# git lets the toolchain stamp the commit into the binary (see buildVersion)
RUN apk add --no-cache ca-certificates git

WORKDIR /build

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /dist/modbot .

FROM scratch

LABEL org.opencontainers.image.source="https://github.com/MemeLabs/modbot" \
      org.opencontainers.image.description="strims.gg chat moderation bot" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /dist/modbot /modbot

# Scraped over the `strims` docker network; deliberately not published to the host.
EXPOSE 9090

# There is no shell or curl in a scratch image, so the binary probes itself.
# Assumes the default -metrics port; override both together if you change it.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
	CMD ["/modbot", "-healthcheck"]

ENTRYPOINT ["/modbot"]
