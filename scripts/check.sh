#!/usr/bin/env bash

set -euo pipefail

REPOSITORY_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd -- "$REPOSITORY_ROOT"

echo "Checking Go formatting..."
unformatted_files="$(gofmt -l cmd internal)"
if [[ -n "$unformatted_files" ]]; then
    echo "Error: these Go files are not formatted:" >&2
    echo "$unformatted_files" >&2
    echo "Run: gofmt -w cmd internal" >&2
    exit 1
fi

echo "Running tests..."
go test ./...

echo "Running go vet..."
go vet ./...

echo "Testing release-note generation..."
./scripts/test-generate-release-notes.sh

echo "Checking the working diff for whitespace errors..."
git --no-pager diff --check HEAD

echo "Checks passed."
