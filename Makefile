BINARY=plugin
GO_FILES=$(shell find . -maxdepth 2 -name '*.go')

.PHONY: build clean test lint fmt

build:
	@echo "Building $(BINARY)..."
	go build -o $(BINARY) main.go

test:
	@echo "Running tests..."
	go test -v ./...

lint:
	@echo "Linting code..."
	golangci-lint run --timeout=5m

fmt:
	@echo "Formatting code..."
	go fmt ./...

clean:
	@echo "Cleaning up..."
	rm -f $(BINARY)
