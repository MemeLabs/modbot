FROM golang:alpine

ENV GO111MODULE=on

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build -o modbot .
WORKDIR /dist

RUN cp /build/modbot .

CMD ["/dist/modbot"]
