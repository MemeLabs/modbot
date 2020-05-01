FROM golang:alpine

ENV GO111MODULE=on

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY *.go ./
RUN go build -o modbot .
WORKDIR /dist

RUN cp /build/modbot .

ENTRYPOINT ["/dist/modbot"]
