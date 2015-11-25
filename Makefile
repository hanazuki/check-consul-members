all: deps
	go build ./...

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
