#!/usr/bin/env bash

set -e

gobin="${GOBIN:-go}"

if ! command -v "golangci-lint" &> /dev/null; then
    echo "MISSING DEPENDENCY: required executable golangci-lint is not available"
    exit 1
fi

echo "linting code..."

"$gobin" vet ./...

golangci-lint run ./...

if [ "$(gofmt -d .)" != "" ]; then
    echo "code is not properly formatted. Fix with 'make format'"
    exit 1
fi

"$gobin" run github.com/bobg/mingo/cmd/mingo@latest -check ./
