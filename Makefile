.PHONY: all build clean install test test-unit test-integration test-coverage

# Build parameters
DARN_BINARY_NAME = darn
DARNIT_BINARY_NAME = darnit
VERSION = dev
COMMIT = $(shell git rev-parse --short HEAD)
BUILD_DIR = build
DARN_MAIN_DIR = cmd/darn
DARNIT_MAIN_DIR = cmd/darnit

# Go parameters
GOCMD = go
GOBUILD = $(GOCMD) build
GOINSTALL = $(GOCMD) install
GOTEST = $(GOCMD) test
GOMOD = $(GOCMD) mod
GOFLAGS = -v
LDFLAGS = -ldflags "-X github.com/kusari-oss/darn/internal/version.Version=$(VERSION) -X github.com/kusari-oss/darn/internal/version.Commit=$(COMMIT)"

# Test parameters
TEST_PACKAGES = ./...
COVERAGE_PROFILE = coverage.out
COVERAGE_HTML = coverage.html

all: clean build
build: build_darn build_darnit

build_darn:
	@echo "Building $(DARN_BINARY_NAME) version $(VERSION) (commit: $(COMMIT))"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(DARN_BINARY_NAME) ./$(DARN_MAIN_DIR)

build_darnit:
	@echo "Building $(DARNIT_BINARY_NAME) version $(VERSION) (commit: $(COMMIT))"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(GOFLAGS) $(LDFLAGS) -o $(BUILD_DIR)/$(DARNIT_BINARY_NAME) ./$(DARNIT_MAIN_DIR)

install: install_darn install_darnit

install_darn:
	@echo "Installing $(DARN_BINARY_NAME) version $(VERSION) (commit: $(COMMIT))"
	$(GOINSTALL) $(GOFLAGS) $(LDFLAGS) ./$(DARN_MAIN_DIR)

install_darnit:
	@echo "Installing $(DARNIT_BINARY_NAME) version $(VERSION) (commit: $(COMMIT))"
	$(GOINSTALL) $(GOFLAGS) $(LDFLAGS) ./$(DARNIT_MAIN_DIR)

# Run all tests
test:
	$(GOTEST) -v $(TEST_PACKAGES)

# Run only unit tests
test-unit:
	$(GOTEST) -v -short $(TEST_PACKAGES)

# Run integration tests
test-integration:
	$(GOTEST) -v -tags=integration -integration $(TEST_PACKAGES)

# Run CLI tests (requires building binaries first)
test-cli: build
	$(GOTEST) -v -tags=cli -cli $(TEST_PACKAGES)

# Run all tests with coverage reporting
test-coverage:
	$(GOTEST) -v -coverprofile=$(COVERAGE_PROFILE) $(TEST_PACKAGES)
	$(GOCMD) tool cover -html=$(COVERAGE_PROFILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated at $(COVERAGE_HTML)"

clean:
	@echo "Cleaning build directory"
	@rm -rf $(BUILD_DIR)
	@go clean
	@rm -f $(COVERAGE_PROFILE) $(COVERAGE_HTML)

tidy:
	$(GOMOD) tidy

# Get test dependencies
test-deps:
	$(GOCMD) get github.com/stretchr/testify/assert
	$(GOCMD) get github.com/stretchr/testify/require
	$(GOCMD) get github.com/stretchr/testify/mock