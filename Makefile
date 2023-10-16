# Makefile adapted from https://github.com/thockin/go-build-template
# Released under the Apache-2.0 license.
DBG_MAKEFILE ?=
ifeq ($(DBG_MAKEFILE),1)
    $(warning ***** starting Makefile for goal(s) "$(MAKECMDGOALS)")
    $(warning ***** $(shell date))
else
    # If we're not debugging the Makefile, don't echo recipes.
    MAKEFLAGS += -s
endif

VERSION ?= "0.0.0-$(shell git log -n 1 --pretty=format:%h 2>/dev/null || printf "0000000")"

OS := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))

DBG ?=

# We don't need make's built-in rules.
MAKEFLAGS += --no-builtin-rules
# Be pedantic about undefined variables.
MAKEFLAGS += --warn-undefined-variables

OS := $(if $(GOOS),$(GOOS),$(shell go env GOOS))
ARCH := $(if $(GOARCH),$(GOARCH),$(shell go env GOARCH))

TAG := $(VERSION)__$(OS)_$(ARCH)

GOFLAGS ?=
HTTP_PROXY ?=
HTTPS_PROXY ?=

default: verify

test: # @HELP run tests
test:
	./scripts/test.sh

lint: # @HELP run code linting
lint:
	./scripts/lint.sh

format: # @HELP format code
format:
	./scripts/format.sh

verify: # @HELP run tests and code linting
verify: lint test

install: # @HELP build and install from current code
install:
	OS=$(OS) ARCH=$(ARCH) VERSION=$(VERSION) ./scripts/install.sh

clean: # @HELP remove build artifacts
clean:
	rm -rf ./bin
	rm -rf ./dist

help: # @HELP prints this message
help:
	echo "VARIABLES:"
	echo "  VERSION  = $(VERSION)"
	echo "  OS       = $(OS)"
	echo "  ARCH     = $(ARCH)"
	echo "  DBG      = $(DBG)"
	echo "  GOFLAGS  = $(GOFLAGS)"
	echo
	echo "TARGETS:"
	grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST)     \
	    | awk '                                   \
	        BEGIN {FS = ": *# *@HELP"};           \
	        { printf "  %-30s %s\n", $$1, $$2 };  \
	    '
