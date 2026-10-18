#!/usr/bin/env bash

set -euo pipefail

usage() {
    echo "Usage: $0 <version> <dev|prod>" >&2
}

if [[ $# -ne 2 ]]; then
    usage
    exit 1
fi

BUILD_VERSION="$1"
BUILD_MODE="$2"

if [[ ! "$BUILD_VERSION" =~ ^[0-9A-Za-z][0-9A-Za-z._+-]*$ ]]; then
    echo "Error: version must contain only letters, numbers, '.', '_', '+', or '-', and must start with a letter or number." >&2
    exit 1
fi

if [[ "$BUILD_MODE" != "dev" && "$BUILD_MODE" != "prod" ]]; then
    echo "Error: build mode must be either 'dev' or 'prod'." >&2
    usage
    exit 1
fi

REPOSITORY_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_DIRECTORY="$REPOSITORY_ROOT/build/linux/amd64"
OUTPUT_PATH="$OUTPUT_DIRECTORY/kstankcli"
COMPLETION_DIRECTORY="$OUTPUT_DIRECTORY/completions"
BASH_COMPLETION_PATH="$COMPLETION_DIRECTORY/kstankcli.bash"
BASH_COMPLETION_TEMP="$BASH_COMPLETION_PATH.tmp"
ZSH_COMPLETION_PATH="$COMPLETION_DIRECTORY/_kstankcli"
ZSH_COMPLETION_TEMP="$ZSH_COMPLETION_PATH.tmp"
LINKER_VALUES="-X github.com/KyleStank/kstankcli/internal/cli.Version=$BUILD_VERSION -X github.com/KyleStank/kstankcli/internal/cli.BuildMode=$BUILD_MODE"

mkdir -p "$OUTPUT_DIRECTORY" "$COMPLETION_DIRECTORY"
cd -- "$REPOSITORY_ROOT"

echo "Building kstankcli $BUILD_VERSION ($BUILD_MODE) for linux/amd64..."

if [[ "$BUILD_MODE" == "dev" ]]; then
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -gcflags="all=-N -l" \
        -ldflags="$LINKER_VALUES" \
        -o "$OUTPUT_PATH" \
        ./cmd/kstankcli
else
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
        -trimpath \
        -ldflags="-s -w $LINKER_VALUES" \
        -o "$OUTPUT_PATH" \
        ./cmd/kstankcli
fi

echo "Built $OUTPUT_PATH"
"$OUTPUT_PATH" --version

trap 'rm -f "$BASH_COMPLETION_TEMP" "$ZSH_COMPLETION_TEMP"' EXIT
"$OUTPUT_PATH" completion bash > "$BASH_COMPLETION_TEMP"
"$OUTPUT_PATH" completion zsh > "$ZSH_COMPLETION_TEMP"
mv -- "$BASH_COMPLETION_TEMP" "$BASH_COMPLETION_PATH"
mv -- "$ZSH_COMPLETION_TEMP" "$ZSH_COMPLETION_PATH"
trap - EXIT

echo "Generated Bash completion at $BASH_COMPLETION_PATH"
echo "Generated Zsh completion at $ZSH_COMPLETION_PATH"
