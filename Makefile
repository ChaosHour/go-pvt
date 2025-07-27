# Makefile for go-pvt

# Variables
BINARY_NAME=go-pvt
BINARY_PATH=bin/$(BINARY_NAME)
SOURCE_PATH=cmd/pvt/main.go

# View formatter variables
VIEW_FORMATTER_BINARY=view-formatter
VIEW_FORMATTER_PATH=bin/$(VIEW_FORMATTER_BINARY)
VIEW_FORMATTER_SOURCE=cmd/view-formatter/main.go

# Default target
all: build build-view-formatter

# Build the binary
build:
	@mkdir -p bin
	go build -o $(BINARY_PATH) $(SOURCE_PATH)

# Build the view formatter
build-view-formatter:
	@mkdir -p bin
	go build -o $(VIEW_FORMATTER_PATH) $(VIEW_FORMATTER_SOURCE)

# Clean build artifacts
clean:
	rm -rf bin/

# Install dependencies
deps:
	go mod tidy

# Run the application (requires parameters)
run:
	go run $(SOURCE_PATH)

# Run the view formatter
run-view-formatter:
	go run $(VIEW_FORMATTER_SOURCE)

# Run view formatter with the prod file
run-view-formatter-prod:
	go run $(VIEW_FORMATTER_SOURCE) prod-views-2b-altered-with-alter-2025-07-15.txt

# Test with first 20 lines
test-view-formatter:
	head -20 prod-views-2b-altered-with-alter-2025-07-15.txt | go run $(VIEW_FORMATTER_SOURCE) /dev/stdin

# Build for multiple platforms
build-all:
	@mkdir -p bin
	GOOS=darwin GOARCH=amd64 go build -o bin/$(BINARY_NAME)-darwin-amd64 $(SOURCE_PATH)
	GOOS=darwin GOARCH=amd64 go build -o bin/$(VIEW_FORMATTER_BINARY)-darwin-amd64 $(VIEW_FORMATTER_SOURCE)

# Help target
help:
	@echo "Available targets:"
	@echo "  build              - Build the go-pvt binary (default)"
	@echo "  build-view-formatter - Build the view-formatter binary"
	@echo "  clean              - Remove build artifacts"
	@echo "  deps               - Install dependencies"
	@echo "  run                - Run the go-pvt application"
	@echo "  run-view-formatter - Run the view-formatter application"
	@echo "  run-view-formatter-prod - Run view-formatter with prod file"
	@echo "  test-view-formatter - Test view formatter with first 20 lines of prod file"
	@echo "  build-all          - Build for multiple platforms"
	@echo "  help               - Show this help message"

.PHONY: all build build-view-formatter clean deps run run-view-formatter run-view-formatter-prod test-view-formatter build-all help
