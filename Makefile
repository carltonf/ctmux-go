BINARY_NAME = ctmux
SOURCE_FILES = .
UPX = upx

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -w -s -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)
GO_BUILD_FLAGS = -trimpath -ldflags="$(LDFLAGS)"

OUT_DIR = out
DEBUG_DIR = $(OUT_DIR)/debug
RELEASE_DIR = $(OUT_DIR)/release

DEBUG_LINUX_BIN = $(DEBUG_DIR)/$(BINARY_NAME)
DEBUG_WIN32_BIN = $(DEBUG_DIR)/$(BINARY_NAME).exe

RELEASE_LINUX_BIN = $(RELEASE_DIR)/$(BINARY_NAME)-linux-amd64
RELEASE_WIN32_BIN = $(RELEASE_DIR)/$(BINARY_NAME)-windows-amd64.exe

# Default target
all: build

# Test and lint
test:
	go test -v ./...

vet:
	go vet ./...

fmt:
	go fmt ./...

# Create output directories
$(DEBUG_DIR) $(RELEASE_DIR):
	@mkdir -p $(DEBUG_DIR) $(RELEASE_DIR)

# Debug builds (Linux and Windows amd64)
build: linux win32

linux: $(DEBUG_DIR)
	@echo "Building Linux binary ($(VERSION))..."
	go build -ldflags="$(LDFLAGS)" -o $(DEBUG_LINUX_BIN) $(SOURCE_FILES)

win32: $(DEBUG_DIR)
	@echo "Building Windows binary ($(VERSION))..."
	GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o $(DEBUG_WIN32_BIN) $(SOURCE_FILES)

# Release builds with optional UPX compression
release: release-linux release-win32

release-linux: $(RELEASE_DIR)
	@echo "Building optimized Linux binary..."
	GOOS=linux GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(RELEASE_LINUX_BIN) $(SOURCE_FILES)
	@if command -v $(UPX) >/dev/null 2>&1; then \
		echo "Compressing with UPX..."; \
		$(UPX) --best --ultra-brute $(RELEASE_LINUX_BIN); \
	else \
		echo "Notice: UPX not found, skipping compression."; \
	fi

release-win32: $(RELEASE_DIR)
	@echo "Building optimized Windows binary..."
	GOOS=windows GOARCH=amd64 go build $(GO_BUILD_FLAGS) -o $(RELEASE_WIN32_BIN) $(SOURCE_FILES)
	@if command -v $(UPX) >/dev/null 2>&1; then \
		echo "Compressing with UPX..."; \
		$(UPX) --best --ultra-brute $(RELEASE_WIN32_BIN); \
	else \
		echo "Notice: UPX not found, skipping compression."; \
	fi

# Local installation
install:
	@echo "Installing ctmux..."
	go install $(GO_BUILD_FLAGS) $(SOURCE_FILES)

# Clean build artifacts
clean:
	@echo "Cleaning up..."
	rm -rf $(OUT_DIR)

.PHONY: all build release linux win32 release-linux release-win32 test vet fmt install clean
