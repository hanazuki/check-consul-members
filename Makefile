all: deps lint
	go build -ldflags "-X main.VERSION=$$(git describe --tags --dirty 2> /dev/null) " ./...

lint: dev-deps
	go vet ./...
	golint ./... | tee .golint.txt && test ! -s .golint.txt

test: lint all
	go test ./...

clean:
	go clean

deps:
	go get -d -v ./...

dev-deps:
	go get github.com/golang/lint/golint

.PHONY: all test lint clean deps dev-deps
