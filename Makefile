all: deps lint build

build: dev-deps
	go build -ldflags "-X main.Version=$$(git describe --tags --dirty 2> /dev/null) " ./...

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

pkg: all
	tar zcf check-consul-members_$$(go env GOOS)_$$(go env GOARCH).tgz check-consul-members

.PHONY: all build test lint clean deps dev-deps pkg
