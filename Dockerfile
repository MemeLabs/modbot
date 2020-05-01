FROM golang:alpine AS builder
RUN apk --no-cache add ca-certificates

ENV GO111MODULE=on \
    CGO_ENABLED=0

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
ENTRYPOINT ["/modbot"]
