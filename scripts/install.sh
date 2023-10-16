#!/usr/bin/env bash

set -e

if [ "${OS:-}" = "" ]; then
    echo "OS must be set"
    exit 1
fi
if [ "${ARCH:-}" = "" ]; then
    echo "ARCH must be set"
    exit 1
fi
if [ "${VERSION:-}" = "" ]; then
    echo "VERSION must be set"
    exit 1
fi

gobin="${GOBIN:-go}"

commit_sha="$(git log -n 1 --pretty=format:%H 2>/dev/null || printf "0000000000000000000000000000000000000000")"
build_time="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
go_version="$(go env GOVERSION)"

export CGO_ENABLED=0
export GOARCH="$ARCH"
export GOOS="$OS"
export GO111MODULE=on

if [[ "${DBG:-}" == 1 ]]; then
    # Debugging - disable optimizations and inlining
    gogcflags="all=-N -l"
    goasmflags=""
    goldflags=""
else
    # Not debugging - trim paths, disable symbols and DWARF.
    goasmflags="all=-trimpath=$PWD"
    gogcflags="all=-trimpath=$PWD"
    goldflags="-s -w"
fi

always_ldflags="-X $(go list -m)/internal/cli.buildVersion=${VERSION} "
always_ldflags+="-X $(go list -m)/internal/cli.buildCommit=${commit_sha} "
always_ldflags+="-X $(go list -m)/internal/cli.buildTime=${build_time} "
always_ldflags+="-X $(go list -m)/internal/cli.buildGoVersion=${go_version}"

echo "building and installing pkgdmp v${VERSION}..."
"$gobin" install \
    -installsuffix "static" \
    -gcflags="$gogcflags" \
    -asmflags="$goasmflags" \
    -ldflags="${always_ldflags} ${goldflags}" \
    ./cmd/pkgdmp/...
