#!/usr/bin/env bash

set -e

fmtcmd="gofmt"

if command -v "gofumpt" &> /dev/null; then
    fmtcmd="gofumpt"
fi

echo "formatting code with $fmtcmd..."
"$fmtcmd" -w .
