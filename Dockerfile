FROM golang:alpine AS builder
RUN apk --no-cache add ca-certificates

ENV CGO_ENABLED=0

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY *.go ./
RUN go build .
WORKDIR /dist
RUN cp /build/modbot .

FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /dist/modbot /

# Scraped over the `strims` docker network; deliberately not published to the host.
EXPOSE 9090

# There is no shell or curl in a scratch image, so the binary probes itself.
# Assumes the default -metrics port; override both together if you change it.
HEALTHCHECK --interval=30s --timeout=5s --start-period=60s --retries=3 \
	CMD ["/modbot", "-healthcheck"]

ENTRYPOINT ["/modbot"]
