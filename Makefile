GIT_SERVER 	:= github.com
ORG			:= Benbentwo
REPO        := go-markdown2confluence
BINARY 		:= markdown2confluence

# Pretty Constant stuff Below, Configurable above

VERSION_REPO := $(GIT_SERVER)/$(ORG)/$(NAME)
# Make does not offer a recursive wildcard function, so here's one:
rwildcard=$(wildcard $1$2) $(foreach d,$(wildcard $1*),$(call rwildcard,$d/,$2))

SHELL := /bin/bash
BUILD_TARGET = build
MAIN_SRC_FILE=main.go
GO := GO111MODULE=on go
GO_NOMOD :=GO111MODULE=off go

GO_VERSION := $(shell $(GO) version | sed -e 's/^[^0-9.]*\([0-9.]*\).*/\1/')
GO_DEPENDENCIES := $(call rwildcard,pkg/,*.go) $(call rwildcard,*.go)

.PHONY: build
VERSION=3.0.0

release-build:
	goreleaser release --snapshot --skip-publish --rm-dist

release-test:
	goreleaser release --skip-publish --rm-dist

release:
	goreleaser release --rm-dist

build: $(GO_DEPENDENCIES) ## Build binary for current OS
	CGO_ENABLED=$(CGO_ENABLED) $(GO) $(BUILD_TARGET) $(BUILDFLAGS) -o build/$(BINARY) $(MAIN_SRC_FILE)
