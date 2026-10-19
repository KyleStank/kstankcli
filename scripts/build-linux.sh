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

REPOSITORY_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
LINUX_OUTPUT_DIRECTORY="$REPOSITORY_ROOT/build/linux/amd64"
LINUX_OUTPUT_PATH="$LINUX_OUTPUT_DIRECTORY/kstankcli"
LINUX_COMPLETION_DIRECTORY="$LINUX_OUTPUT_DIRECTORY/completions"
LINUX_BASH_COMPLETION_PATH="$LINUX_COMPLETION_DIRECTORY/kstankcli.bash"
LINUX_BASH_COMPLETION_TEMP="$LINUX_BASH_COMPLETION_PATH.tmp"
LINUX_ZSH_COMPLETION_PATH="$LINUX_COMPLETION_DIRECTORY/_kstankcli"
LINUX_ZSH_COMPLETION_TEMP="$LINUX_ZSH_COMPLETION_PATH.tmp"
WINDOWS_OUTPUT_DIRECTORY="$REPOSITORY_ROOT/build/windows/amd64"
WINDOWS_OUTPUT_PATH="$WINDOWS_OUTPUT_DIRECTORY/kstankcli.exe"
WINDOWS_COMPLETION_DIRECTORY="$WINDOWS_OUTPUT_DIRECTORY/completions"
WINDOWS_BASH_COMPLETION_PATH="$WINDOWS_COMPLETION_DIRECTORY/kstankcli.bash"
WINDOWS_BASH_COMPLETION_TEMP="$WINDOWS_BASH_COMPLETION_PATH.tmp"
WINDOWS_POWERSHELL_COMPLETION_PATH="$WINDOWS_COMPLETION_DIRECTORY/kstankcli.ps1"
WINDOWS_POWERSHELL_COMPLETION_TEMP="$WINDOWS_POWERSHELL_COMPLETION_PATH.tmp"
LINKER_VALUES="-X github.com/KyleStank/kstankcli/internal/cli.Version=$BUILD_VERSION -X github.com/KyleStank/kstankcli/internal/cli.BuildMode=$BUILD_MODE"

mkdir -p \
    "$LINUX_COMPLETION_DIRECTORY" \
    "$WINDOWS_COMPLETION_DIRECTORY"
cd -- "$REPOSITORY_ROOT"

build_target() {
    local target_os="$1"
    local output_path="$2"

    echo "Building kstankcli $BUILD_VERSION ($BUILD_MODE) for $target_os/amd64..."

    if [[ "$BUILD_MODE" == "dev" ]]; then
        CGO_ENABLED=0 GOOS="$target_os" GOARCH=amd64 go build \
            -gcflags="all=-N -l" \
            -ldflags="$LINKER_VALUES" \
            -o "$output_path" \
            ./cmd/kstankcli
    else
        CGO_ENABLED=0 GOOS="$target_os" GOARCH=amd64 go build \
            -trimpath \
            -ldflags="-s -w $LINKER_VALUES" \
            -o "$output_path" \
            ./cmd/kstankcli
    fi

    echo "Built $output_path"
}

build_target linux "$LINUX_OUTPUT_PATH" &
linux_build_pid=$!
build_target windows "$WINDOWS_OUTPUT_PATH" &
windows_build_pid=$!

linux_build_status=0
windows_build_status=0
if ! wait "$linux_build_pid"; then
    linux_build_status=1
fi
if ! wait "$windows_build_pid"; then
    windows_build_status=1
fi

if ((linux_build_status != 0 || windows_build_status != 0)); then
    exit 1
fi

"$LINUX_OUTPUT_PATH" --version

trap 'rm -f "$LINUX_BASH_COMPLETION_TEMP" "$LINUX_ZSH_COMPLETION_TEMP" "$WINDOWS_BASH_COMPLETION_TEMP" "$WINDOWS_POWERSHELL_COMPLETION_TEMP"' EXIT
"$LINUX_OUTPUT_PATH" completion bash > "$LINUX_BASH_COMPLETION_TEMP"
"$LINUX_OUTPUT_PATH" completion zsh > "$LINUX_ZSH_COMPLETION_TEMP"
"$LINUX_OUTPUT_PATH" completion bash > "$WINDOWS_BASH_COMPLETION_TEMP"
"$LINUX_OUTPUT_PATH" completion powershell > "$WINDOWS_POWERSHELL_COMPLETION_TEMP"
mv -- "$LINUX_BASH_COMPLETION_TEMP" "$LINUX_BASH_COMPLETION_PATH"
mv -- "$LINUX_ZSH_COMPLETION_TEMP" "$LINUX_ZSH_COMPLETION_PATH"
mv -- "$WINDOWS_BASH_COMPLETION_TEMP" "$WINDOWS_BASH_COMPLETION_PATH"
mv -- "$WINDOWS_POWERSHELL_COMPLETION_TEMP" "$WINDOWS_POWERSHELL_COMPLETION_PATH"
trap - EXIT

echo "Generated Bash completion at $LINUX_BASH_COMPLETION_PATH"
echo "Generated Zsh completion at $LINUX_ZSH_COMPLETION_PATH"
echo "Generated Git Bash completion at $WINDOWS_BASH_COMPLETION_PATH"
echo "Generated PowerShell completion at $WINDOWS_POWERSHELL_COMPLETION_PATH"
