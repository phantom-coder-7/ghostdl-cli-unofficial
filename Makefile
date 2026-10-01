.PHONY: build build-common build-all clean test lint help install

# Application info
APP_NAME    := ghostdl-unofficial
HOMEPAGE    := https://github.com/phantom-coder-7/ghostdl-cli-unofficial
GIT_TAG     := $(shell git describe --tags --exact-match 2>/dev/null)
GIT_VER     := $(patsubst v%,%,$(patsubst V%,%,$(GIT_TAG)))
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME  := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
DIRTY       := $(if $(shell git status --porcelain 2>/dev/null),true,false)
VERSION_PKG := github.com/phantom-coder-7/ghostdl-cli-unofficial/internal/version

# Build parameters
GO          := go
LDFLAGS     := -X $(VERSION_PKG).ver=$(GIT_VER) -X $(VERSION_PKG).commit=$(COMMIT) -X $(VERSION_PKG).buildDate=$(BUILD_TIME) -X $(VERSION_PKG).dirty=$(DIRTY)
GOFLAGS     := -ldflags="$(LDFLAGS)"
OUTPUT_DIR  := bin

# Default build
build:
	@mkdir -p $(OUTPUT_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -o $(OUTPUT_DIR)/$(APP_NAME) .

ANDROID_API ?= 21

COMMON_TARGETS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

TARGETS := linux/386 linux/amd64 linux/arm linux/arm64 linux/riscv64 \
	darwin/amd64 darwin/arm64 \
	windows/386 windows/amd64 windows/arm64 \
	android/arm android/arm64 android/386 android/amd64

define cross_build_loop
set -e; \
ndk_host=$$(uname -s | tr '[:upper:]' '[:lower:]')-$$(uname -m); \
for target in $(1); do \
	os=$${target%/*}; \
	arch=$${target#*/}; \
	out=$(OUTPUT_DIR)/$(APP_NAME)-$$os-$$arch; \
	if [ "$$arch" = "arm" ]; then out=$(OUTPUT_DIR)/$(APP_NAME)-$$os-armv7; fi; \
	if [ "$$os" = "windows" ]; then out=$$out.exe; fi; \
	if [ "$$os" = "android" ]; then \
		ndk=; \
		for cand in "$${ANDROID_NDK_HOME}" "$${ANDROID_NDK_ROOT}" "$${ANDROID_NDK}" "$${ANDROID_NDK_LATEST_HOME}"; do \
			if [ -n "$$cand" ] && [ -d "$$cand/toolchains/llvm/prebuilt" ]; then ndk=$$cand; break; fi; \
		done; \
		if [ -z "$$ndk" ]; then \
			for sdk in "$${ANDROID_HOME}" "$${ANDROID_SDK_ROOT}"; do \
				if [ -z "$$sdk" ] || [ ! -d "$$sdk/ndk" ]; then continue; fi; \
				ver=$$(ls -1 "$$sdk/ndk" 2>/dev/null | sort -V | tail -1); \
				if [ -n "$$ver" ] && [ -d "$$sdk/ndk/$$ver/toolchains/llvm/prebuilt" ]; then ndk=$$sdk/ndk/$$ver; break; fi; \
			done; \
		fi; \
		llvm=$${ndk}/toolchains/llvm/prebuilt/$${ndk_host}; \
		case "$$arch" in \
			arm64) triple=aarch64-linux-android$(ANDROID_API);; \
			amd64) triple=x86_64-linux-android$(ANDROID_API);; \
			386) triple=i686-linux-android$(ANDROID_API);; \
			arm) triple=armv7a-linux-androideabi$(ANDROID_API);; \
			*) echo "unsupported android arch: $$arch" >&2; exit 1;; \
		esac; \
		cc=$${llvm}/bin/$${triple}-clang; \
		cxx=$${llvm}/bin/$${triple}-clang++; \
		if [ -z "$$ndk" ] || [ ! -d "$$llvm" ] || [ ! -x "$$cc" ]; then \
			echo "Android NDK clang not found; set ANDROID_NDK_HOME to an NDK with toolchains/llvm/prebuilt" >&2; \
			exit 1; \
		fi; \
		if [ "$$arch" = "arm" ]; then \
			CGO_ENABLED=1 CC=$$cc CXX=$$cxx GOOS=$$os GOARCH=$$arch GOARM=7 $(GO) build $(GOFLAGS) -o $$out .; \
		else \
			CGO_ENABLED=1 CC=$$cc CXX=$$cxx GOOS=$$os GOARCH=$$arch $(GO) build $(GOFLAGS) -o $$out .; \
		fi; \
	elif [ "$$arch" = "arm" ]; then \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch GOARM=7 $(GO) build $(GOFLAGS) -o $$out .; \
	else \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build $(GOFLAGS) -o $$out .; \
	fi; \
done
endef

# Cross-platform build (everyday OS/arch for CI)
build-common:
	@mkdir -p $(OUTPUT_DIR)
	@$(call cross_build_loop,$(COMMON_TARGETS))

# Cross-platform build (every release target)
build-all:
	@mkdir -p $(OUTPUT_DIR)
	@$(call cross_build_loop,$(TARGETS))

# Tests
test:
	CGO_ENABLED=1 $(GO) test -v -race -coverprofile=coverage.out ./...

# Code linting
lint:
	golangci-lint run ./...

# Clean up
clean:
	rm -rf $(OUTPUT_DIR) coverage.out

# Install to GOPATH
install:
	$(GO) install $(GOFLAGS) .

# Help
help:
	@echo "Usage:"
	@echo "  make build        — Build binary for current platform"
	@echo "  make build-common — Cross-compile everyday OS/arch for CI"
	@echo "  make build-all    — Cross-compile every release target"
	@echo "  make test       — Run tests"
	@echo "  make lint       — Run code linting"
	@echo "  make clean      — Clean build artifacts"
	@echo "  make install    — Install to GOPATH"
