.PHONY: fmt test build install tidy

fmt:
	gofmt -w main.go internal/provider/*.go

test:
	go test ./...

build:
	go build ./...

install:
	go install

tidy:
	go mod tidy
