.PHONY: build generate schema install deps test cover

BINARY_NAME := linear

deps:
	go mod tidy

schema:
	curl -sSL -o schema.graphql \
		https://raw.githubusercontent.com/linear/linear/master/packages/sdk/src/schema.graphql

generate:
	go generate ./...

build: generate
	go build -o $(BINARY_NAME) .

install: generate
	go install .

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
