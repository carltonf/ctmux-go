BINARY_NAME = ctmux
SOURCE_FILES = .
UPX = upx
GO_BUILD_FLAGS = -a -gcflags=all="-l -B" -ldflags="-w -s"

WIN32_GOOS = windows
WIN32_GOARCH = amd64 # Change to 386 for 32-bit

# Default to build a debug version
all: build

build: linux win32
release: release-linux release-win32

release-linux: linux-opt
release-win32: win32-opt

# Create output directories {
OUT_DIR = out
DEBUG_DIR = $(OUT_DIR)/debug
RELEASE_DIR = $(OUT_DIR)/release
# Create necessary directories before building
$(DEBUG_DIR) $(RELEASE_DIR):
	@mkdir -p $(DEBUG_DIR) $(RELEASE_DIR)
# }

DEBUG_LINUX_BIN = $(DEBUG_DIR)/$(BINARY_NAME)
DEBUG_WIN32_BIN = $(DEBUG_DIR)/$(BINARY_NAME).exe
RELEASE_LINUX_BIN = $(RELEASE_DIR)/$(BINARY_NAME)
RELEASE_WIN32_BIN = $(RELEASE_DIR)/$(BINARY_NAME).exe

linux: $(DEBUG_DIR)
	@echo "Building the binary ..."
	go build -o $(DEBUG_LINUX_BIN) $(SOURCE_FILES)
linux-opt: $(RELEASE_DIR)
	@echo "Building the binary with optimizations..."
	go build $(GO_BUILD_FLAGS) -o $(RELEASE_LINUX_BIN) $(SOURCE_FILES)
	@echo "Compressing the binary with UPX..."
	$(UPX) --best --ultra-brute $(RELEASE_LINUX_BIN)

# Build for Win32
win32: $(DEBUG_DIR)
	@echo "Building the binary for Win32..."
	GOOS=$(WIN32_GOOS) GOARCH=$(WIN32_GOARCH) go build -o $(DEBUG_WIN32_BIN) $(SOURCE_FILES)
win32-opt: $(RELEASE_DIR)
	@echo "Building the binary for Win32 with optimizations..."
	GOOS=$(WIN32_GOOS) GOARCH=$(WIN32_GOARCH) go build $(GO_BUILD_FLAGS) -o $(RELEASE_WIN32_BIN) $(SOURCE_FILES)
	@echo "Compressing the win32 binary with UPX..."
	$(UPX) --best --ultra-brute $(RELEASE_WIN32_BIN)

# Clean up generated files
clean:
	@echo "Cleaning up..."
	rm -rf $(OUT_DIR)

.PHONY: all build release linux linux-opt win32 win32-opt clean
