BINARY_NAME=viberay
CMD_PATH=./cmd/viberay
BUILD_DIR=./build

.PHONY: all build clean test lint fmt vet run

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean -testcache

test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

test-short:
	@echo "Running short tests..."
	go test -v -short ./...

bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

lint:
	@echo "Linting..."
	@golangci-lint run ./... || true

fmt:
	@echo "Formatting..."
	go fmt ./...

vet:
	@echo "Vetting..."
	go vet ./...

coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

run: build
	$(BUILD_DIR)/$(BINARY_NAME)

# Cross-compilation
build-linux:
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)

build-mac:
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)

build-windows:
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)
