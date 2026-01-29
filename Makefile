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
GOPATH = $(HOME)/go/bin
export PATH := ${PATH}:$(GOPATH)
MODULE_BINARY = find-webcams

ifeq ($(TARGET_OS),windows)
	MODULE_BINARY = find-webcams.exe
	LDFLAGS = -ldflags="-extldflags=-static -extldflags=-static-libgcc -extldflags=-static-libstdc++"
endif

build: format
	rm -f $(BIN_OUTPUT_PATH)/$(MODULE_BINARY)
	CGO_ENABLED=1 go build $(LDFLAGS) -o $(BIN_OUTPUT_PATH)/$(MODULE_BINARY) main.go

module.tar.gz: build
	rm -f bin/module.tar.gz
	cp $(BIN_OUTPUT_PATH)/$(MODULE_BINARY) bin/$(MODULE_BINARY)
	tar czf bin/module.tar.gz bin/$(MODULE_BINARY) meta.json
	rm bin/$(MODULE_BINARY)

setup:
	if [ "$(SOURCE_OS)" = "linux" ]; then \
		sudo apt-get install -y apt-utils coreutils tar libnlopt-dev libjpeg-dev pkg-config; \
	fi
	# remove unused imports
	go install golang.org/x/tools/cmd/goimports@latest
	find . -name '*.go' -exec sh -c '"$$(go env GOPATH)/bin/goimports" -w "$$@"' _ {} +


clean:
	rm -rf $(BIN_OUTPUT_PATH)/$(MODULE_BINARY) bin/module.tar.gz $(MODULE_BINARY)

format:
	gofmt -w -s .
