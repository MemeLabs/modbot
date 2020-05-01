FROM golang:alpine AS builder

ENV GO111MODULE=on

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build .
WORKDIR /dist
RUN cp /build/modbot .

FROM scratch
COPY --from=builder /dist/modbot /
ENTRYPOINT ["/modbot"]
