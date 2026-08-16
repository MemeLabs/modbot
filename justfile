golangci_version := "v2.12.2"

_default:
    @just --list

# build the binary
build:
    go build -trimpath -o modbot .

# run the full check suite, as CI does
check: fmt vet test lint tidy

fmt:
    gofmt -l -w .

vet:
    go vet ./...

test:
    go test -race -count=1 ./...

cover:
    go test -race -count=1 -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

lint:
    go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@{{golangci_version}} run ./...

tidy:
    go mod tidy -diff

# apply available dependency updates
bump:
    go get -u ./...
    go mod tidy

# build the container image the same way CI does
image tag="modbot:dev":
    docker build -t {{tag}} .

# run against chat in read-only mode; needs a jwt cookie
run cookie="":
    go run . -logonly -cookie "{{cookie}}"
