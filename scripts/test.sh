#!/usr/bin/env bash

set -e

gobin="${GOBIN:-go}"
testflags=("-shuffle=on" "-race" "-cover")

if [[ "${DBG:-}" == 1 ]]; then
    testflags+=("-v")
fi

if [[ "${UPDATE_GOLDEN:-}" == 1 ]]; then
    testflags+=("-update-golden")
fi

echo "running tests..."
"$gobin" test "${testflags[@]}" ./...
