BINARY_NAME := linear

.PHONY: deps
deps:
	go mod tidy

.PHONY: schema
schema:
	curl -sSL -o schema.graphql \
		https://raw.githubusercontent.com/linear/linear/master/packages/sdk/src/schema.graphql

.PHONY: generate
generate: schema
	go generate ./...

.PHONY: build
build: generate
	go build -o $(BINARY_NAME) .

.PHONY: install
install: generate
	go install .

.PHONY: test
test:
	go test -race ./...

.PHONY: cover
cover:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
