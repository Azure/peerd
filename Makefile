# Compiler variables.
VERSION = $$(git rev-parse --short HEAD)

# Go build variables.
GOCMD = go
GOTEST = $(GOCMD) test
GOBUILD = $(GOCMD) build -installsuffix 'static' -ldflags "-X main.version=$(VERSION)"
GOLINT = golangci-lint run

# Source repository variables.
ROOT_DIR := $(shell git rev-parse --show-toplevel)
BIN_DIR = $(ROOT_DIR)/bin
TEST_PKGS = $(shell go list ./...)
TESTS_BIN_DIR = $(BIN_DIR)/tests
COVERAGE_DIR=$(BIN_DIR)/coverage
SCRIPTS_DIR=$(ROOT_DIR)/scripts

include $(ROOT_DIR)/build/ci/Makefile
include $(ROOT_DIR)/tests/Makefile

.DEFAULT_GOAL := all

.PHONY: all
all: lint test build ## Runs the peerd build targets in the correct order

.PHONY: build
build: ## Build the peerd packages
	@echo "+ $@"
	@( $(GOBUILD) -o $(BIN_DIR)/peerd ./cmd/proxy )

.PHONY: install
install: build ## Installs the peerd service in the project bin directory
	@echo "+ $@"
	@( cp $(ROOT_DIR)/init/systemd/peerd.service $(BIN_DIR)/peerd.service )
	@( cp $(ROOT_DIR)/api/swagger.yaml $(BIN_DIR)/swagger.yaml )

.PHONY: help
help: info ## Generates help for all targets with a description.
# Read the makefile and print out all targets that have a comment after them.
# If external Makefiles are referenced, trim the external reference from the target name. ex. Makefile:help: -> help:
# Sort the output.
# Split the string based on the Field Separator (FS) and print the first and second fields.
	@grep -E '^[^#[:space:]].*?## .*$$' $(MAKEFILE_LIST) | sed -E 's/^[^:]+:([^:]+:)/\1/' | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: install-gocov
install-gocov: ## Install Go cov.
	@echo "+ $@"
	@( go install github.com/axw/gocov/gocov@latest && \
    	go install gotest.tools/gotestsum@latest && \
    	go install github.com/jandelgado/gcov2lcov@latest && \
    	go install github.com/AlekSi/gocov-xml@latest )

.PHONY: install-linter
install-linter: ## Install Go linter.
	@echo "+ $@"
	@( curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /usr/local/bin v1.54.2 )

.PHONY: build-image
build-image: ## Build the peerd docker image
	@echo "+ $@"
ifndef CONTAINER_REGISTRY
	$(eval CONTAINER_REGISTRY := $(shell echo "localhost"))
endif
	$(call build-image-internal,$(ROOT_DIR)/build/package/Dockerfile,peerd,$(ROOT_DIR))

info: header

.PHONY: lint
lint: ## Run linter.
	@echo "+ $@"
	@( $(GOLINT) --timeout=10m ./... )

.PHONY: test
test: ## Runs tests.
	@echo "+ $@"
	@( $(GOTEST) ./... )

.PHONY: coverage
coverage: ## Generates test results for code coverage
	@echo "+ $@"
	@( COVERAGE_DIR=$(COVERAGE_DIR) $(SCRIPTS_DIR)/coverage.sh "$(ROOT_DIR)" "$(TEST_PKGS)" true )

.PHONY: swag
swag: ## Generates the swagger documentation of the p2p server.
	@echo "+ $@"
	cd $(ROOT_DIR)/internal/handlers; swag init --ot go,yaml -o $(ROOT_DIR)/api -g ./root.go

.PHONY: add-copyright
add-copyright: ## Add the copyright header to all Go files.
	@echo "+ $@"
	find . -type f -name "*.go" -exec sh -c 'grep -q -F "// Copyright (c) Microsoft Corporation." "$0" || sed -i "1i\\// Copyright (c) Microsoft Corporation.\\n// Licensed under the Apache License, Version 2.0." "$0"' {} \;

define HEADER

	 _____	                _
	|  __ \                | |
	| |__) |__  ___ _ __ __| |
	|  ___/ _ \/ _ \ '__/ _` |
	| |  |  __/  __/ | | (_| |
	|_|   \___|\___|_|  \__,_|
                         
endef

export HEADER

header:
	@echo "$$HEADER"

# build-image-internal takes the dockerfile location, repository name and build context.
# Example: 
define build-image-internal
	@echo "\033[92mBuilding Image: $2\033[0m"

	@echo docker build -f $1 \
	-t localhost/$2:dev \
	$3

	@docker build -f $1 \
	-t localhost/$2:dev \
	$3
endef
