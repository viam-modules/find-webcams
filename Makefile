SOURCE_OS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
SOURCE_ARCH ?= $(shell uname -m)
TARGET_OS ?= $(SOURCE_OS)
TARGET_ARCH ?= $(SOURCE_ARCH)
normalize_arch = $(if $(filter aarch64,$(1)),arm64,$(if $(filter x86_64,$(1)),amd64,$(1)))
# Normalize the source and target arch to arm64 or amd64 for compatibility with go build.
SOURCE_ARCH := $(call normalize_arch,$(SOURCE_ARCH))
TARGET_ARCH := $(call normalize_arch,$(TARGET_ARCH))
TOOL_BIN = bin/gotools/$(shell uname -s)-$(shell uname -m)
BIN_OUTPUT_PATH = bin/$(TARGET_OS)-$(TARGET_ARCH)
export PATH := ${PATH}:$(shell go env GOPATH)/bin
MODULE_BINARY = find-webcams

# Set cross-compilation environment based on TARGET_OS
GO_BUILD_ENV :=
ifeq ($(TARGET_OS),windows)
	GO_BUILD_ENV += GOOS=windows GOARCH=$(TARGET_ARCH) CC=x86_64-w64-mingw32-gcc
	MODULE_BINARY = find-webcams.exe
else ifeq ($(TARGET_OS),linux)
	GO_BUILD_ENV += GOOS=linux GOARCH=$(TARGET_ARCH)
else ifeq ($(TARGET_OS),darwin)
	GO_BUILD_ENV += GOOS=darwin GOARCH=$(TARGET_ARCH)
endif

build: format update-rdk
	rm -f $(BIN_OUTPUT_PATH)/$(MODULE_BINARY)
	$(GO_BUILD_ENV) CGO_ENABLED=1 go build $(LDFLAGS) -o $(BIN_OUTPUT_PATH)/$(MODULE_BINARY) main.go

module.tar.gz: build
	rm -f bin/module.tar.gz
	cp $(BIN_OUTPUT_PATH)/$(MODULE_BINARY) bin/$(MODULE_BINARY)
	tar czf bin/module.tar.gz bin/$(MODULE_BINARY) meta.json
	rm bin/$(MODULE_BINARY)

setup:
	if [ "$(SOURCE_OS)" = "linux" ]; then \
		sudo apt-get install -y apt-utils coreutils tar libnlopt-dev libjpeg-dev pkg-config gcc-mingw-w64-x86-64; \
	fi
	# remove unused imports
	go install golang.org/x/tools/cmd/goimports@latest
	find . -name '*.go' -exec sh -c '"$$(go env GOPATH)/bin/goimports" -w "$$1"' _ {} \;


clean:
	rm -rf $(BIN_OUTPUT_PATH)/find-webcams bin/module.tar.gz find-webcams

format:
	gofmt -w -s .
	
update-rdk:
	go get go.viam.com/rdk@latest
	go mod tidy
